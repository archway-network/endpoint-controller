package blockchain_test

import (
	"encoding/json"
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

func createTestServer(handler http.Handler) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestHandleGetRequests(t *testing.T) {
	// Create the first test server
	ts1 := createTestServer(http.HandlerFunc(handleGetRequest1))
	defer ts1.Close()

	// Create the second test server
	ts2 := createTestServer(http.HandlerFunc(handleGetRequest2))
	defer ts2.Close()

	// Create the third test server
	ts3 := createTestServer(http.HandlerFunc(handleGetRequest3))
	defer ts3.Close()

	healthy := []string{
		ts1.Listener.Addr().String(),
		ts2.Listener.Addr().String(),
		ts3.Listener.Addr().String(),
	}
	expectedHealthy := []string{ts1.Listener.Addr().String(), ts2.Listener.Addr().String()}

	blockchain.CheckNodeBehind(&healthy, 6)

	assert.Equal(t, expectedHealthy, healthy)
}
