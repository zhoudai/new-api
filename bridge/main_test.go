package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestBridgeRequiresProjectAndEnterpriseContext(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer upstream-key" {
			t.Fatalf("bridge must replace project key before forwarding")
		}
		if request.Header.Get(enterpriseIDHeader) != "school-a" || request.Header.Get(featureHeader) != "analysis" {
			t.Fatalf("bridge must forward billing context")
		}
		if request.Header.Get(platformRequestIDHeader) != "project-request-1" {
			t.Fatalf("bridge must adapt the project request id for New API")
		}
		response.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	upstreamURL, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	config := bridgeConfig{
		upstreamBaseURL:  upstreamURL,
		projectID:        "demo-project",
		projectKey:       "project-key",
		enterpriseTokens: map[string]string{"school-a": "upstream-key"},
	}
	handler := newBridgeHandler(config)

	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	request.Header.Set(projectIDHeader, "demo-project")
	request.Header.Set("Authorization", "Bearer project-key")
	request.Header.Set(featureHeader, "analysis")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("missing enterprise id should return 400, got %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	request.Header.Set(projectIDHeader, "demo-project")
	request.Header.Set(enterpriseIDHeader, "school-a")
	request.Header.Set(featureHeader, "analysis")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing project key should return 401, got %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	request.Header.Set(projectIDHeader, "demo-project")
	request.Header.Set("Authorization", "Bearer project-key")
	request.Header.Set(enterpriseIDHeader, "school-a")
	request.Header.Set(featureHeader, "analysis")
	request.Header.Set(projectRequestIDHeader, "project-request-1")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("mapped enterprise request should be forwarded, got %d", response.Code)
	}
}

func TestParseEnterpriseTokensRejectsDuplicateIdentifiers(t *testing.T) {
	if _, err := parseEnterpriseTokens("school-a=one\nschool-a=two"); err == nil {
		t.Fatal("duplicate enterprise identifiers must be rejected")
	}
}
