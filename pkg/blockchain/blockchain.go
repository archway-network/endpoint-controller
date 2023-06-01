package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/archway-network/endpoint-controller/pkg/utils"
)

const (
	httpTimeout = 5
)

type NodeStatus struct {
	Result struct {
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
		} `json:"sync_info"`
	} `json:"result"`
}

func getRequest(host string, path string) ([]byte, error) {
	cli := &http.Client{
		Timeout: httpTimeout * time.Second,
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+host+path, nil)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	resp, err := cli.Do(req)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func checkOpenPorts(host string, ports []corev1.EndpointPort) error {
	timeout := httpTimeout * time.Second

	for _, port := range ports {
		klog.Infof("checking node %s port %d protocol %s", host, port.Port, port.Protocol)
		if _, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(int(port.Port))), timeout); err != nil {
			return fmt.Errorf(
				"could not get correct answer from %s:%d, marking target unhealthy",
				host,
				port.Port)
		}
	}
	return nil
}

func CheckNodeBehind(healthy *[]string, blockMiss int) {
	var highest int
	var hostPort string
	nodeBlockheights := make(map[string]int)

	// loop every healthy target to check if the node is falling behind
	for _, ip := range *healthy {
		klog.Infof("checking node %s block height", ip)
		var nodeStatus NodeStatus

		// to make this function testable
		if strings.Contains(ip, "127.0.0.1") {
			hostPort = ip
		} else {
			hostPort = net.JoinHostPort(ip, "26657")
		}

		// get the status REST call and get the latest block height
		data, err := getRequest(hostPort, "/status")
		if err != nil {
			klog.Error(err)
			continue
		}
		err = json.Unmarshal(data, &nodeStatus)
		if err != nil {
			klog.Error(err)
			continue
		}

		blockHeightInt, err := strconv.Atoi(nodeStatus.Result.SyncInfo.LatestBlockHeight)
		if err != nil {
			klog.Error(err)
			continue
		}

		// set new highest block height if needed
		if blockHeightInt > highest {
			highest = blockHeightInt
		}
		nodeBlockheights[ip] = blockHeightInt
	}

	// compare block heights
	// remove target from healthy if highest is greater than blockmiss amount
	// loop this until there are no more unhealthy endpoints
	for k, v := range nodeBlockheights {
		if (highest - v) > blockMiss {
			*healthy = utils.RemoveFromSlice(*healthy, k)
		}
	}
}

func HealthCheck(ips []string, ports []corev1.EndpointPort, blockMiss int) []string {
	var healthy []string
	for _, ip := range ips {
		klog.Infof("checking blockchain node (%s) health", ip)
		if err := checkOpenPorts(ip, ports); err != nil {
			klog.Error(err)
			continue
		}
		healthy = append(healthy, ip)
	}
	CheckNodeBehind(&healthy, blockMiss)
	return healthy
}
