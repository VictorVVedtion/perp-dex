package e2e

// helpers_test.go - Common test helpers for E2E tests
// These tests require API server to be running

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// API base URL - requires server to be running
const apiBase = "http://localhost:8080"

// HTTP client with timeout
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// httpGet performs a GET request and returns JSON response
func httpGet(t *testing.T, url string) map[string]interface{} {
	t.Helper()

	resp, err := httpClient.Get(url)
	require.NoError(t, err, "GET %s failed", url)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	require.NoError(t, err, "Failed to parse JSON response: %s", string(body))

	return result
}

// checkAPIAvailable skips the test if API server is not running
func checkAPIAvailable(t *testing.T) {
	t.Helper()
	resp, err := http.Get(apiBase + "/health")
	if err != nil {
		t.Skipf("API server not available at %s: %v", apiBase, err)
	}
	resp.Body.Close()
}

// httpPost performs a POST request with JSON body and returns response
func httpPost(t *testing.T, url string, data map[string]string) map[string]interface{} {
	t.Helper()

	jsonData, err := json.Marshal(data)
	require.NoError(t, err, "Failed to marshal request body")

	resp, err := httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err, "POST %s failed", url)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	require.NoError(t, err, "Failed to parse JSON response: %s", string(body))

	return result
}
