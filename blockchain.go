package main

import (
	"context"
	"net"
	"net/http"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func responsiveEndpoint(ip string, port int32) bool {
	hostPort := net.JoinHostPort(ip, strconv.Itoa(int(port)))
	cli := &http.Client{}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+hostPort, nil) // OK
	if err != nil {
		klog.Error(err)
		return false
	}
	resp, err := cli.Do(req)
	resp.Body.Close()
	if err != nil {
		klog.Error(err)
		return false
	}
	return true
}

func blockchainHealthCheck(ip string, ports []corev1.EndpointPort) bool {
	klog.Infof("checking blockchain node (%s) health", ip)
	for _, port := range ports {
		klog.Infof("checking node %s port %d protocol %s", ip, port.Port, port.Protocol)
		if !responsiveEndpoint(ip, port.Port) {
			klog.Errorf("Could not get correct answer from %s:%d, marking target unhealthy", ip, port.Port)
			return false
		}
	}
	return true
}
