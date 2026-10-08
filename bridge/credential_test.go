package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedCredentialBridgeFailsClosed(t *testing.T) {
	status := http.StatusNoContent
	verifier := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer managed-key", r.Header.Get("Authorization"))
		assert.Equal(t, "project-a", r.Header.Get(projectIDHeader))
		w.WriteHeader(status)
	}))
	defer verifier.Close()
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, "Bearer internal-key", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	upstreamURL, err := url.Parse(upstream.URL)
	require.NoError(t, err)
	handler := newBridgeHandler(bridgeConfig{projectID: "project-a", projectKey: "managed-key", credentialVerifyURL: verifier.URL, upstreamBaseURL: upstreamURL, enterpriseTokens: map[string]string{"enterprise-a": "internal-key"}})
	for _, tc := range []struct{ status, expected int }{{204, 204}, {401, 401}, {503, 503}, {302, 503}} {
		status = tc.status
		request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		request.Header.Set("Authorization", "Bearer managed-key")
		request.Header.Set(projectIDHeader, "project-a")
		request.Header.Set(enterpriseIDHeader, "enterprise-a")
		request.Header.Set(featureHeader, "analysis")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		assert.Equal(t, tc.expected, response.Code)
	}
	assert.Equal(t, 1, calls, "禁用、故障和重定向均不得回退到环境变量密钥")
}
