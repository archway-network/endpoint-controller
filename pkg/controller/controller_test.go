package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/archway-network/endpoint-controller/pkg/controller"
)

// func TestMain(m *testing.M) {
// 	klog.SetOutput(ioutil.Discard)
// 	os.Exit(m.Run())
// }

func TestController(t *testing.T) {
	// create a fake clientset
	clientset := fake.NewSimpleClientset()

	// create a test service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
			Annotations: map[string]string{
				"endpoint-controller/enable":  "true",
				"endpoint-controller/targets": "1.1.1.1,2.2.2.2",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "test-port",
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
				{
					Name:       "test-port2",
					Port:       8081,
					TargetPort: intstr.FromInt(8081),
				},
				{
					Name:       "test-port3",
					Port:       8082,
					TargetPort: intstr.FromInt(8082),
				},
			},
		},
	}

	// create a test endpoint
	endpoint := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
			Annotations: map[string]string{
				"endpoint-controller/enable": "true",
			},
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.1.1.1",
					},
					{
						IP: "2.2.2.2",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name: "test-port",
						Port: 8080,
					},
					{
						Name: "test-port2",
						Port: 8081,
					},
					{
						Name: "test-port3",
						Port: 8082,
					},
				},
			},
		},
	}

	// add the test service to the fake clientset
	_, err := clientset.CoreV1().
		Services(service.Namespace).
		Create(context.Background(), service, metav1.CreateOptions{})
	assert.NoError(t, err)

	// create a new controller
	c := controller.Controller{
		Clientset: clientset,
		Resync:    time.Duration(1) * time.Second,
	}

	// start the controller
	go c.Run()

	//nolint: staticcheck // the wait package we are using does not have PollWithContextTimeout
	err = wait.PollImmediate(1*time.Second, 6*time.Second, func() (bool, error) {
		_, err = clientset.CoreV1().Endpoints(
			endpoint.Namespace).Get(context.Background(),
			endpoint.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	// check that the endpoint is correct
	actualEndpoint, err := clientset.CoreV1().Endpoints(
		endpoint.Namespace).Get(context.Background(),
		endpoint.Name, metav1.GetOptions{})

	assert.Equal(t, endpoint, actualEndpoint)
}

func TestUpdateEndpoint(t *testing.T) {
	// create a fake clientset
	clientset := fake.NewSimpleClientset()

	// create a new controller
	c := controller.Controller{
		Clientset: clientset,
		Resync:    time.Duration(1) * time.Second,
	}

	// create a test endpoint
	endpoint := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
			Annotations: map[string]string{
				"endpoint-controller/enable": "true",
			},
		}, Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.1.1.1",
					},
					{
						IP: "2.2.2.2",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name: "test-port",
						Port: 8080,
					},
					{
						Name: "test-port2",
						Port: 8081,
					},
					{
						Name: "test-port3",
						Port: 8082,
					},
				},
			},
		},
	}
	// add the test endpoint to the fake clientset
	_, err := clientset.CoreV1().Endpoints(
		endpoint.Namespace).Create(context.Background(),
		endpoint, metav1.CreateOptions{})

	assert.NoError(t, err)
	ips := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}
	err = c.UpdateEndpointTargets(*endpoint, ips)
	assert.NoError(t, err)

	expectedEndpoint := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
			Annotations: map[string]string{
				"endpoint-controller/enable": "true",
			},
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.1.1.1",
					},
					{
						IP: "2.2.2.2",
					},
					{
						IP: "3.3.3.3",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name: "test-port",
						Port: 8080,
					},
					{
						Name: "test-port2",
						Port: 8081,
					},
					{
						Name: "test-port3",
						Port: 8082,
					},
				},
			},
		},
	}

	// check that the endpoint is correct
	actualEndpoint, err := clientset.CoreV1().Endpoints(
		endpoint.Namespace).Get(context.Background(),
		endpoint.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	assert.Equal(t, expectedEndpoint, actualEndpoint)
}

func TestRemoveEndpointTarget(t *testing.T) {
	// create a fake clientset
	clientset := fake.NewSimpleClientset()

	// create a new controller
	c := controller.Controller{
		Clientset: clientset,
		Resync:    time.Duration(1) * time.Second,
	}

	// create a test endpoint
	endpoint := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
			Annotations: map[string]string{
				"endpoint-controller/enable": "true",
			},
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.1.1.1",
					},
					{
						IP: "2.2.2.2",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name: "test-port",
						Port: 8080,
					},
					{
						Name: "test-port2",
						Port: 8081,
					},
					{
						Name: "test-port3",
						Port: 8082,
					},
				},
			},
		},
	}
	expectedEndpoint := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
			Annotations: map[string]string{
				"endpoint-controller/enable": "true",
			},
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.1.1.1",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name: "test-port",
						Port: 8080,
					},
					{
						Name: "test-port2",
						Port: 8081,
					},
					{
						Name: "test-port3",
						Port: 8082,
					},
				},
			},
		},
	}
	// add the test service to the fake clientset
	_, err := clientset.CoreV1().Endpoints(
		endpoint.Namespace).Create(context.Background(),
		endpoint, metav1.CreateOptions{})
	assert.NoError(t, err)

	ips := []string{"1.1.1.1"}
	err = c.UpdateEndpointTargets(*endpoint, ips)
	assert.NoError(t, err)

	// check that the endpoint is correct
	actualEndpoint, err := clientset.CoreV1().Endpoints(
		endpoint.Namespace).Get(context.Background(),
		endpoint.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	assert.Equal(t, expectedEndpoint, actualEndpoint)
}
