package main //nolint:cyclop // complexity is not a big issue for now

import (
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
		klog.Fatal(err)
	}

	syncPeriod = time.Duration(syncPeriodEnv) * time.Second

	blockMiss, err = utils.GetEnv("BLOCK_MISS", defaultBlockMiss)
	if err != nil {
		klog.Fatal(err)
	}

	// create the Kubernetes client object using the service account
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatal(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err.Error())
	}

	// create a controller to handle service events
	c := controller.Controller{
		Clientset: clientset,
		Resync:    syncPeriod,
		BlockMiss: blockMiss,
	}

	// start the controller
	c.Run()
}
