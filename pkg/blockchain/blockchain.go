package blockchain

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/strings/slices"

	"github.com/archway-network/endpoint-controller/pkg/utils"
)

type NodeStatus struct {
	Result struct {
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
		} `json:"sync_info"`
	} `json:"result"`
}

func getRequest(host string, path string) ([]byte, error) {
	cli := &http.Client{}
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

func checkNodeBehind(healthy, unhealthy []string, blockMiss int) ([]string, []string) {
	var highest int
	nodeBlockheights := make(map[string]int)
	for _, ip := range healthy {
		klog.Infof("checking node %s block height", ip)
		var nodeStatus NodeStatus
		hostPort := net.JoinHostPort(ip, "26657")
		data, err := getRequest(hostPort, "/status")
		if err != nil {
			klog.Error(err)
			unhealthy = append(unhealthy, ip)
			continue
		}
		err = json.Unmarshal(data, &nodeStatus)
		if err != nil {
			klog.Error(err)
			unhealthy = append(unhealthy, ip)
			continue
		}

		blockHeightInt, err := strconv.Atoi(nodeStatus.Result.SyncInfo.LatestBlockHeight)
		if err != nil {
			klog.Error(err)
			unhealthy = append(unhealthy, ip)
			continue
		}

		if blockHeightInt > highest {
			highest = blockHeightInt
		}
		nodeBlockheights[ip] = blockHeightInt
		healthy = append(healthy, ip)
	}

	// compare block heights
	// remove target from healthy if highest is greater than blockmiss amount
	for k, v := range nodeBlockheights {
		if (highest - v) >= blockMiss {
			unhealthy = append(unhealthy, k)
			healthy = utils.RemoveFromSlice(healthy, k)
		}
	}
	return healthy, unhealthy
}

func BlockchainHealthCheck(ips []string, ports []corev1.EndpointPort, blockMiss int) ([]string, []string) {
	var healthy, unhealthy []string
	for _, ip := range ips {
		klog.Infof("checking blockchain node (%s) health", ip)
		for _, port := range ports {
			klog.Infof("checking node %s port %d protocol %s", ip, port.Port, port.Protocol)
			hostPort := net.JoinHostPort(ip, strconv.Itoa(int(port.Port)))
			if _, err := getRequest(hostPort, "/"); err != nil {
				klog.Errorf("Could not get correct answer from %s:%d, marking target unhealthy", ip, port.Port)
				unhealthy = append(unhealthy, ip)
				break
			}
		}
		if !slices.Contains(unhealthy, ip) {
			healthy = append(healthy, ip)
		}
	}

	return checkNodeBehind(healthy, unhealthy, blockMiss)
}
