package blockchain_test

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/archway-network/endpoint-controller/pkg/blockchain"
)

func handleGetRequest1(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := blockchain.NodeStatus{}
	response.Result.SyncInfo.LatestBlockHeight = "1000"
	_ = json.NewEncoder(w).Encode(response)
}

func handleGetRequest2(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := blockchain.NodeStatus{}
	response.Result.SyncInfo.LatestBlockHeight = "1002"
	_ = json.NewEncoder(w).Encode(response)
}

func handleGetRequest3(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := blockchain.NodeStatus{}
	response.Result.SyncInfo.LatestBlockHeight = "992"
	_ = json.NewEncoder(w).Encode(response)
}

func createTestServer(addr string, handler http.Handler) (*httptest.Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	ts := httptest.NewUnstartedServer(handler)
	ts.Listener = l
	ts.Start()

	return ts, nil
}

func TestHandleGetRequests(t *testing.T) {
	healthy := []string{"127.0.0.1:26657", "127.0.0.1:26658", "127.0.0.1:26659"}
	expectedHealthy := []string{"127.0.0.1:26657", "127.0.0.1:26658"}
	// Create the first test server
	ts1, err := createTestServer("127.0.0.1:26657", http.HandlerFunc(handleGetRequest1))
	if err != nil {
		t.Fatal(err)
	}
	defer ts1.Close()

	// Create the second test server
	ts2, err := createTestServer("127.0.0.1:26658", http.HandlerFunc(handleGetRequest2))
	if err != nil {
		t.Fatal(err)
	}
	defer ts2.Close()

	// Create the third test server
	ts3, err := createTestServer("127.0.0.1:26659", http.HandlerFunc(handleGetRequest3))
	if err != nil {
		t.Fatal(err)
	}
	defer ts3.Close()

	blockchain.CheckNodeBehind(&healthy, 6)

	assert.Equal(t, expectedHealthy, healthy)
}
