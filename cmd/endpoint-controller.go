package main //nolint:cyclop // complexity is not a big issue for now

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

	"github.com/archway-network/endpoint-controller/pkg/controller"
	"github.com/archway-network/endpoint-controller/pkg/utils"
)

const (
	defaultSyncPeriod = "30"
	defaultBlockMiss  = "6"
)

func main() {
	var syncPeriod time.Duration
	var blockMiss int

	// Get environment variables, if not use defaults
	syncPeriodEnv, err := utils.GetEnv("SYNC_PERIOD", defaultSyncPeriod)
	if err != nil {
		panic(err)
	}

	syncPeriod = time.Duration(syncPeriodEnv) * time.Second

	blockMiss, err = utils.GetEnv("BLOCK_MISS", defaultBlockMiss)
	if err != nil {
		panic(err)
	}

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

	// create a workqueue to handle service events
	workqueueConfig := workqueue.RateLimitingQueueConfig{
		Name: "services",
	}
	queue := workqueue.NewRateLimitingQueueWithConfig(workqueue.DefaultControllerRateLimiter(), workqueueConfig)

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
	controller := controller.Controller{
		Clientset: clientset,
		Queue:     queue,
		Recorder:  recorder,
		Resync:    syncPeriod,
		BlockMiss: blockMiss,
	}

	// start the controller
	go controller.Run()

	//nolint: exhaustive // we don't need all the watch parameters here
	// loop through each event and add the service to the workqueue
	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Added, watch.Modified:
			service, _ := event.Object.(*corev1.Service)
			if service.Annotations["endpoint-controller/enabled"] == "true" {
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
