package main

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

const (
	defaultResyncPeriod = 30 * time.Second
)

func main() {
	// create the Kubernetes client object using the service account
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// create a watch interface for Kubernetes services
	watcher, err := clientset.CoreV1().Services("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	klog.Infof("Watching services...")

	// create a workqueue to handle service events
	queue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "services")

	// create an event recorder to log events
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: clientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		corev1.EventSource{
			Component: "endpoint-controller",
		},
	)

	// create a controller to handle service events
	controller := &Controller{
		clientset: clientset,
		queue:     queue,
		recorder:  recorder,
		resync:    defaultResyncPeriod,
	}

	// start the controller
	go controller.Run()

	//nolint: exhaustive // we don't need all the watch parameters here
	// loop through each event and add the service to the workqueue
	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Added, watch.Modified:
			service, _ := event.Object.(*corev1.Service)
			if service.Annotations["endpoint-controller-enabled"] == "true" {
				queue.Add(service)
			}
		case watch.Error:
			klog.Fatal("something bad happened during watch")
		case watch.Deleted:
			// handle service deletion here
			klog.Infof("Service deleted")
		}
	}
}
