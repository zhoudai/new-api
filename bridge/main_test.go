package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBridgeTLSCertificateAndKeyMustBeConfiguredTogether(t *testing.T) {
	baseEnv := map[string]string{
		"AI_BRIDGE_UPSTREAM_BASE_URL":     "http://new-api:3000",
		"AI_BRIDGE_PROJECT_ID":            "demo-project",
		"AI_BRIDGE_CREDENTIAL_VERIFY_URL": "http://new-api:3000/api/project-credentials/verify",
		"AI_BRIDGE_ENTERPRISE_TOKENS":     "school-a=upstream-key",
	}

	for _, tlsEnv := range []map[string]string{
		{},
		{"AI_BRIDGE_TLS_CERT_FILE": "/tls/fullchain.pem"},
		{"AI_BRIDGE_TLS_KEY_FILE": "/tls/privkey.pem"},
	} {
		t.Run(tlsEnv["AI_BRIDGE_TLS_CERT_FILE"]+tlsEnv["AI_BRIDGE_TLS_KEY_FILE"], func(t *testing.T) {
			values := make(map[string]string, len(baseEnv)+len(tlsEnv))
			for key, value := range baseEnv {
				values[key] = value
			}
			for key, value := range tlsEnv {
				values[key] = value
			}
			_, err := loadConfigFromEnv(func(key string) string { return values[key] })
			require.ErrorContains(t, err, "禁止 HTTP 回退")
		})
	}
}

func TestBridgeServerUsesBoundedReadAndIdleTimeouts(t *testing.T) {
	server := newBridgeServer(bridgeConfig{listenAddr: ":3001"})
	require.Equal(t, 120*time.Second, server.IdleTimeout)
	require.Equal(t, 5*time.Minute, server.ReadTimeout)
	require.Equal(t, 1<<20, server.MaxHeaderBytes)
	require.Zero(t, server.WriteTimeout)
}

func TestBridgeRejectsInvalidTLSKeyPair(t *testing.T) {
	missingCert := filepath.Join(t.TempDir(), "missing.crt")
	values := map[string]string{
		"AI_BRIDGE_TLS_CERT_FILE":         missingCert,
		"AI_BRIDGE_TLS_KEY_FILE":          "/tls/privkey.pem",
		"AI_BRIDGE_UPSTREAM_BASE_URL":     "http://new-api:3000",
		"AI_BRIDGE_PROJECT_ID":            "demo-project",
		"AI_BRIDGE_CREDENTIAL_VERIFY_URL": "http://new-api:3000/api/project-credentials/verify",
		"AI_BRIDGE_ENTERPRISE_TOKENS":     "school-a=upstream-key",
	}
	_, err := loadConfigFromEnv(func(key string) string { return values[key] })
	require.ErrorContains(t, err, "Bridge TLS 证书或私钥无法加载")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestBridgeServerUsesPreloadedTLSCertificate(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer upstream.Close()

	certificate := upstream.TLS.Certificates[0]
	keyBytes, err := x509.MarshalPKCS8PrivateKey(certificate.PrivateKey)
	require.NoError(t, err)
	tlsDir := t.TempDir()
	certFile := filepath.Join(tlsDir, "cert.pem")
	keyFile := filepath.Join(tlsDir, "key.pem")
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Certificate[0]}), 0o600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}), 0o600))

	values := map[string]string{
		"AI_BRIDGE_TLS_CERT_FILE":         certFile,
		"AI_BRIDGE_TLS_KEY_FILE":          keyFile,
		"AI_BRIDGE_UPSTREAM_BASE_URL":     "http://new-api:3000",
		"AI_BRIDGE_PROJECT_ID":            "demo-project",
		"AI_BRIDGE_CREDENTIAL_VERIFY_URL": "http://new-api:3000/api/project-credentials/verify",
		"AI_BRIDGE_ENTERPRISE_TOKENS":     "school-a=upstream-key",
	}
	config, err := loadConfigFromEnv(func(key string) string { return values[key] })
	require.NoError(t, err)

	server := newBridgeServer(config)
	require.NotNil(t, server.TLSConfig.GetCertificate)
	servedCertificate, err := server.TLSConfig.GetCertificate(&tls.ClientHelloInfo{})
	require.NoError(t, err)
	require.Same(t, config.tlsCertificate, servedCertificate)
	require.Equal(t, certificate.Certificate, servedCertificate.Certificate)
}

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
