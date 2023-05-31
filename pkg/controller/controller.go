package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"

	"github.com/archway-network/endpoint-controller/pkg/blockchain"
)

const (
	EndpointControllerEnable  = "endpoint-controller/enable"
	EndpointControllerTargets = "endpoint-controller/targets"
)

// Controller defines the endpoint controller.
type Controller struct {
	Clientset kubernetes.Interface
	Resync    time.Duration
	BlockMiss int
}

// Run starts the endpoint controller.
func (c *Controller) Run() {
	klog.Info("Starting endpoint controller...")

	// set up the resync timer
	timer := time.NewTicker(c.Resync)
	defer timer.Stop()
	klog.Infof("Synching every %s", c.Resync)

	c.resyncEndpoints()
	for range timer.C {
		klog.Info("Resynching endpoints")
		c.resyncEndpoints()
	}
}

// checkPortSync checks ports are matching between service and endpoint.
func (c *Controller) checkPortSync(service corev1.Service, endpoints corev1.Endpoints) bool {
	serviceEndpointPortObjects := createEndpointPortObject(service)

	if len(serviceEndpointPortObjects) != len(endpoints.Subsets[0].Ports) {
		return false
	}

	for i, x := range serviceEndpointPortObjects {
		if x != endpoints.Subsets[0].Ports[i] {
			return false
		}
	}

	return true
}

// create endpoint port object.
func createEndpointPortObject(service corev1.Service) []corev1.EndpointPort {
	var ports []corev1.EndpointPort
	for _, port := range service.Spec.Ports {
		ports = append(ports, corev1.EndpointPort{
			Name: port.Name, Protocol: port.Protocol, Port: port.Port,
		})
	}
	return ports
}

// create endpoint addresses object.
func createEndpointAddressObject(service corev1.Service) ([]corev1.EndpointAddress, error) {
	var addresses []corev1.EndpointAddress
	serviceEndpointsAddress := service.Annotations[EndpointControllerTargets]
	if serviceEndpointsAddress == "" {
		return addresses, fmt.Errorf(
			"%s : annotation endpoint-controller/targets is empty",
			service.Name,
		)
	}

	for _, address := range strings.Split(serviceEndpointsAddress, ",") {
		addresses = append(addresses, corev1.EndpointAddress{
			IP: strings.TrimSpace(address),
		})
	}
	return addresses, nil
}

// patchEndpoints patches the endpoint with correct set of data.
func (c *Controller) patchEndpoints(endpoints corev1.Endpoints) error {
	_, err := c.Clientset.CoreV1().Endpoints(endpoints.Namespace).Update(
		context.Background(), &endpoints, v1.UpdateOptions{},
	)
	if err != nil {
		return err
	}

	klog.Infof("Updated endpoint %s", endpoints.Name)
	return nil
}

// createEndpoints creates an endpoint for the given service.
func (c *Controller) createEndpoints(service corev1.Service) error {
	var err error
	var subset corev1.EndpointSubset

	retryErr := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		endpoints := &corev1.Endpoints{
			ObjectMeta: v1.ObjectMeta{
				Name:        service.Name,
				Namespace:   service.Namespace,
				Annotations: map[string]string{EndpointControllerEnable: "true"},
			},
			Subsets: []corev1.EndpointSubset{},
		}

		// create subset objects.
		subset.Ports = createEndpointPortObject(service)
		subset.Addresses, err = createEndpointAddressObject(service)
		if err != nil {
			return err
		}
		endpoints.Subsets = append(endpoints.Subsets, subset)

		// create the endpoints.
		_, err = c.Clientset.CoreV1().Endpoints(service.Namespace).Create(
			context.Background(), endpoints, v1.CreateOptions{},
		)
		if err != nil {
			return err
		}

		klog.Infof("Created endpoint for service %s\n", service.Name)

		return nil
	})

	return retryErr
}

// resyncEndpoints updates all endpoints for the watched services.
func (c *Controller) resyncEndpoints() {
	services, err := c.Clientset.CoreV1().Services("").List(context.Background(), v1.ListOptions{})
	if err != nil {
		klog.Error(err)
	}

	// check all services that have operator enabled.
	for _, service := range services.Items {
		if service.Annotations[EndpointControllerEnable] == "true" {
			if err = c.findEndpoints(service); err != nil {
				klog.Error(err)
			}
		}
	}

	klog.Info("Finished synching endpoints")
}

// findEndpoints
// finds endpoints and checks if it matches with the service
// if it matches, checks the endpoints targets health
// if not found, creates the endpoints
// return error if something breaks.
func (c *Controller) findEndpoints(service corev1.Service) error {
	endpoints, err := c.Clientset.CoreV1().
		Endpoints(service.Namespace).
		Get(context.Background(), service.Name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			if err = c.createEndpoints(service); err != nil {
				return err
			}
		}
		return err
	}

	return c.checkEndpoints(service, *endpoints)
}

// check if endpoint exists and the configuration is up to date
// return error if nothing goes wrong.
func (c *Controller) checkEndpoints(service corev1.Service, endpoint corev1.Endpoints) error {
	if !c.checkPortSync(service, endpoint) {
		endpoint.Subsets[0].Ports = createEndpointPortObject(service)
		if err := c.patchEndpoints(endpoint); err != nil {
			return err
		}
	}

	// split the annotation and remove spaces
	ips := strings.Split(service.Annotations[EndpointControllerTargets], ",")
	for ip := range ips {
		ips[ip] = strings.TrimSpace(ips[ip])
	}
	healthyTargets := blockchain.HealthCheck(
		ips,
		endpoint.Subsets[0].Ports,
		c.BlockMiss,
	)

	// check if the endpoint target will matches the healthy ones
	klog.Infof("Healthy Targets: %v", healthyTargets)
	klog.Infof("Endpoint Targets: %v", endpoint.Subsets[0].Addresses)
	if len(healthyTargets) != len(endpoint.Subsets[0].Addresses) {
		return c.UpdateEndpointTargets(endpoint, healthyTargets)
	}
	for _, ht := range healthyTargets {
		if !checkEndpointsIP(ht, endpoint) {
			if err := c.UpdateEndpointTargets(endpoint, healthyTargets); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkEndpointsIP
// return true if target can be found from endpoints
// return false if target cannot be found.
func checkEndpointsIP(ip string, endpoints corev1.Endpoints) bool {
	for _, address := range endpoints.Subsets[0].Addresses {
		if address.IP == ip {
			return true
		}
	}
	return false
}

// Update endpoint targets.
func (c *Controller) UpdateEndpointTargets(endpoints corev1.Endpoints, ips []string) error {
	endpointAddressList := []corev1.EndpointAddress{}
	for _, ip := range ips {
		endpointAddressList = append(endpointAddressList, corev1.EndpointAddress{
			IP: ip,
		})
		endpoints.Subsets[0].Addresses = endpointAddressList
	}

	klog.Infof("resynching endpoints (%s) targets (%s)", endpoints.Name, ips)
	return c.patchEndpoints(endpoints)
}
