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
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
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
	Queue     workqueue.RateLimitingInterface
	Resync    time.Duration
	Recorder  record.EventRecorder
	BlockMiss int
}

// Run starts the endpoint controller.
func (c *Controller) Run() {
	defer c.Queue.ShutDown()

	klog.Info("Starting endpoint controller...")

	// set up the resync timer
	timer := time.NewTicker(c.Resync)
	defer timer.Stop()
	klog.Infof("Synching every %s", c.Resync)

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
			IP: address,
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

		// c.Recorder.Eventf(
		// 	&service,
		// 	corev1.EventTypeNormal,
		// 	"CreatedEndpoint", "Created endpoint for service %s", service.Name,
		// )
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
		Endpoints("").
		Get(context.Background(), service.Name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return c.createEndpoints(service)
		}
		return err
	}

	return c.checkEndpoints(service, *endpoints)
}

// check if endpoint exists and the configuration is up to date
// return true if everything is OK or if error is found
// returns false if endpoint needs to be created.
func (c *Controller) checkEndpoints(service corev1.Service, endpoint corev1.Endpoints) error {
	if !c.checkPortSync(service, endpoint) {
		endpoint.Subsets[0].Ports = createEndpointPortObject(service)
		if err := c.patchEndpoints(endpoint); err != nil {
			return err
		}
	}

	ips := strings.Split(service.Annotations[EndpointControllerTargets], ",")
	healthyTarget, unhealthyTarget := blockchain.HealthCheck(
		ips,
		endpoint.Subsets[0].Ports,
		c.BlockMiss,
	)

	// add target to endpoints if it does not already exists
	for _, ht := range healthyTarget {
		if !checkEndpointsIP(ht, endpoint) {
			if err := c.AddEndpointTarget(endpoint, ht); err != nil {
				return err
			}
		}
	}

	// remove unhealthy target from endpoints
	for _, ut := range unhealthyTarget {
		if err := c.RemoveEndpointTarget(endpoint, ut); err != nil {
			return err
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

// AddEndpointTarget
// adds target IP from endpoint.
func (c *Controller) AddEndpointTarget(endpoints corev1.Endpoints, ip string) error {
	newEndpointAddress := corev1.EndpointAddress{
		IP: ip,
	}
	endpoints.Subsets[0].Addresses = append(endpoints.Subsets[0].Addresses, newEndpointAddress)

	klog.Infof("adding healthy target %s to endpoint %s", ip, endpoints.Name)
	return c.patchEndpoints(endpoints)
}

// RemoveEndpointTarget
// removes target IP from endpoint.
func (c *Controller) RemoveEndpointTarget(endpoints corev1.Endpoints, ip string) error {
	var endpointAddresses []corev1.EndpointAddress
	const minimumNumber = 2

	// Return if target cannot be found from endpoint.
	if !checkEndpointsIP(ip, endpoints) {
		return nil
	}

	// Remove target from endpoint but fail if it's the last one since endpoints
	// cannot be empty.
	klog.Infof("Removing endpoints (%s) target %s", endpoints.Name, ip)
	if len(endpoints.Subsets[0].Addresses) < minimumNumber {
		klog.Warningf("Cannot remove the last IP in endpoint %s", endpoints.Name)
		return nil
	}

	for _, address := range endpoints.Subsets[0].Addresses {
		if address.IP == ip {
			continue
		}
		endpointAddresses = append(endpointAddresses, address)
	}
	endpoints.Subsets[0].Addresses = endpointAddresses

	klog.Infof("removing unhealthy target %s from endpoint %s", ip, endpoints.Name)
	return c.patchEndpoints(endpoints)
}
