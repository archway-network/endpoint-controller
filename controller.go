package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// Controller defines the endpoint controller.
type Controller struct {
	clientset kubernetes.Interface
	queue     workqueue.RateLimitingInterface
	resync    time.Duration
	recorder  record.EventRecorder
}

// Run starts the endpoint controller.
func (c *Controller) Run() {
	defer c.queue.ShutDown()

	klog.Info("Starting endpoint controller...")

	// set up the resync timer
	timer := time.NewTicker(c.resync)
	quit := make(chan struct{})
	defer timer.Stop()
	klog.Infof("Synching every %s", c.resync)

	for {
		select {
		case <-timer.C:
			klog.Info("Resyncing endpoints...")
			c.resyncEndpoints()
			timer.Reset(c.resync)
		case <-quit:
			timer.Stop()
			return
		}
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
	serviceEndpointsAddress := service.Annotations["endpoint-controller-addresses"]
	if serviceEndpointsAddress == "" {
		return addresses, fmt.Errorf("%s : annotation endpoint-controller-addresses is empty", service.Name)
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
	_, err := c.clientset.CoreV1().Endpoints(endpoints.Namespace).Update(
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
				Annotations: map[string]string{"endpoint-controller-enable": "true"},
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
		_, err = c.clientset.CoreV1().Endpoints(service.Namespace).Create(
			context.Background(), endpoints, v1.CreateOptions{},
		)
		if err != nil {
			return err
		}

		c.recorder.Eventf(
			&service,
			corev1.EventTypeNormal,
			"CreatedEndpoint", "Created endpoint for service %s", service.Name,
		)
		klog.Infof("Created endpoint for service %s\n", service.Name)

		return nil
	})

	return retryErr
}

// TODO: loop throug endpoints that have correct annotation and check if the service exists for that
// if not then delete, if yes don't do anything
// deleteEndpoints deletes the endpoint for the given service
// func (c *Controller) deleteEndpoints(service *corev1.Service) error {
// 	err := c.clientset.CoreV1().Endpoints(service.Namespace).Delete(
// 	context.Background(), service.Name, v1.DeleteOptions{}
// 	)
// 	if err != nil {
// 		return err
// 	}
//
// 	c.recorder.Eventf(
// 	service,
// 	corev1.EventTypeNormal,
// 	"DeletedEndpoint", "Deleted endpoint for service %s", service.Name,
// 	)
// 	klog.Infof("Deleted endpoint for service %s\n", service.Name)
//
// 	return nil
// }

// resyncEndpoints updates all endpoints for the watched services.
func (c *Controller) resyncEndpoints() {
	services, err := c.clientset.CoreV1().Services("").List(context.Background(), v1.ListOptions{})
	if err != nil {
		klog.Error(err)
	}

	// check all services that have operator enabled.
	for _, service := range services.Items {
		if service.Annotations["endpoint-controller-enable"] == "true" {
			if err = c.findEndpoints(service); err != nil {
				klog.Error(err)
			}
		}
	}

	klog.Info("Finished synching endpoints")
}

func (c *Controller) findEndpoints(service corev1.Service) error {
	endpoints, err := c.clientset.CoreV1().Endpoints("").List(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	// Check if endpoint exists
	// if exists check health status of endpoint targets
	// if not create the endpoint.
	for _, endpoint := range endpoints.Items {
		if endpoint.Name == service.Name {
			klog.Infof("found existing endpoint %s for service %s \n", endpoint.Name, service.Name)
			if err = c.checkEndpoints(service, endpoint); err != nil {
				return err
			}
		}
	}
	return nil
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

	// Check each IP in service Annotations
	// if target is healthy and not in endpoint - add it
	// if target is not healthy and in endpoint - remove it.
	for _, ip := range strings.Split(service.Annotations["endpoint-controller-addresses"], ",") {
		if blockchainHealthCheck(ip, endpoint.Subsets[0].Ports) {
			if !checkEndpointsIP(ip, endpoint) {
				if err := c.addEndpointTarget(endpoint, ip); err != nil {
					return err
				}
			}
			continue
		}

		if checkEndpointsIP(ip, endpoint) {
			if err := c.removeEndpointTarget(endpoint, ip); err != nil {
				return err
			}
			continue
		}
		return nil
	}

	// create endpoint.
	return c.createEndpoints(service)
}

// checkEndpointsIP.
func checkEndpointsIP(ip string, endpoints corev1.Endpoints) bool {
	for _, address := range endpoints.Subsets[0].Addresses {
		if address.IP == ip {
			return true
		}
	}
	return false
}

// addEndpointTarget
// adds target IP from endpoint.
func (c *Controller) addEndpointTarget(endpoints corev1.Endpoints, ip string) error {
	newEndpointAddress := corev1.EndpointAddress{
		IP: ip,
	}
	endpoints.Subsets[0].Addresses = append(endpoints.Subsets[0].Addresses, newEndpointAddress)

	klog.Infof("adding healthy target %s to endpoint %s", ip, endpoints.Name)
	return c.patchEndpoints(endpoints)
}

// removeEndpointTarget
// removes target IP from endpoint.
func (c *Controller) removeEndpointTarget(endpoints corev1.Endpoints, ip string) error {
	var endpointAddresses []corev1.EndpointAddress
	const minimumNumber = 2

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
