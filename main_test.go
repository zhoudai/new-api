package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/fs"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInternalTLSServer(t *testing.T) {
	certFile, keyFile := writeInternalTLSKeyPair(t)
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	t.Run("disabled without TLS settings", func(t *testing.T) {
		server, err := newInternalTLSServer(handler, func(string) string { return "" })
		require.NoError(t, err)
		assert.Nil(t, server)
	})

	t.Run("loads certificate and enforces TLS 1.2 minimum", func(t *testing.T) {
		values := map[string]string{
			"AI_PLATFORM_INTERNAL_TLS_CERT_FILE": certFile,
			"AI_PLATFORM_INTERNAL_TLS_KEY_FILE":  keyFile,
		}
		handler := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.WriteHeader(http.StatusNoContent)
		})
		server, err := newInternalTLSServer(handler, func(name string) string { return values[name] })
		require.NoError(t, err)
		require.NotNil(t, server)
		assert.Equal(t, ":3001", server.Addr)
		assert.Equal(t, 10*time.Second, server.ReadHeaderTimeout)
		assert.Equal(t, 120*time.Second, server.IdleTimeout)
		assert.Equal(t, uint16(tls.VersionTLS12), server.TLSConfig.MinVersion)
		assert.Len(t, server.TLSConfig.Certificates, 1)

		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		serverErr := make(chan error, 1)
		go func() { serverErr <- server.ServeTLS(listener, "", "") }()
		t.Cleanup(func() {
			_ = server.Close()
			require.ErrorIs(t, <-serverErr, http.ErrServerClosed)
		})

		certificatePEM, err := os.ReadFile(certFile)
		require.NoError(t, err)
		trustedCertificates := x509.NewCertPool()
		require.True(t, trustedCertificates.AppendCertsFromPEM(certificatePEM))
		client := &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				RootCAs:    trustedCertificates,
				ServerName: "new-api.al-bei.cn",
			}},
		}
		defer client.CloseIdleConnections()
		response, err := client.Get("https://" + listener.Addr().String() + "/health/ready")
		require.NoError(t, err)
		defer response.Body.Close()
		assert.Equal(t, http.StatusNoContent, response.StatusCode)
		assert.GreaterOrEqual(t, response.TLS.Version, uint16(tls.VersionTLS12))
	})

	t.Run("rejects an incomplete certificate pair", func(t *testing.T) {
		values := map[string]string{"AI_PLATFORM_INTERNAL_TLS_CERT_FILE": certFile}
		_, err := newInternalTLSServer(handler, func(name string) string { return values[name] })
		require.ErrorContains(t, err, "必须同时配置")
	})

	t.Run("honors a custom listen address", func(t *testing.T) {
		values := map[string]string{
			"AI_PLATFORM_INTERNAL_TLS_LISTEN_ADDR": "127.0.0.1:9443",
			"AI_PLATFORM_INTERNAL_TLS_CERT_FILE":   certFile,
			"AI_PLATFORM_INTERNAL_TLS_KEY_FILE":    keyFile,
		}
		server, err := newInternalTLSServer(handler, func(name string) string { return values[name] })
		require.NoError(t, err)
		require.NotNil(t, server)
		assert.Equal(t, "127.0.0.1:9443", server.Addr)
	})

	t.Run("rejects a key-only certificate pair", func(t *testing.T) {
		values := map[string]string{"AI_PLATFORM_INTERNAL_TLS_KEY_FILE": keyFile}
		_, err := newInternalTLSServer(handler, func(name string) string { return values[name] })
		require.ErrorContains(t, err, "必须同时配置")
	})

	t.Run("preserves certificate loading errors", func(t *testing.T) {
		missingCert := filepath.Join(t.TempDir(), "missing.crt")
		values := map[string]string{
			"AI_PLATFORM_INTERNAL_TLS_CERT_FILE": missingCert,
			"AI_PLATFORM_INTERNAL_TLS_KEY_FILE":  keyFile,
		}
		_, err := newInternalTLSServer(handler, func(name string) string { return values[name] })
		require.Error(t, err)
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
}

func TestShutdownHTTPServersDrainsConcurrently(t *testing.T) {
	requestsStarted := make(chan struct{}, 2)
	releaseRequests := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseRequests) }) }
	defer release()

	newServer := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			requestsStarted <- struct{}{}
			<-releaseRequests
			response.WriteHeader(http.StatusNoContent)
		}))
	}
	publicServer := newServer()
	defer publicServer.Close()
	internalServer := newServer()
	defer internalServer.Close()

	shutdownStarted := make(chan struct{}, 2)
	publicServer.Config.RegisterOnShutdown(func() { shutdownStarted <- struct{}{} })
	internalServer.Config.RegisterOnShutdown(func() { shutdownStarted <- struct{}{} })

	type requestResult struct {
		status int
		err    error
	}
	requestResults := make(chan requestResult, 2)
	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	for _, server := range []*httptest.Server{publicServer, internalServer} {
		go func(url string) {
			response, err := client.Get(url)
			if err != nil {
				requestResults <- requestResult{err: err}
				return
			}
			defer response.Body.Close()
			requestResults <- requestResult{status: response.StatusCode}
		}(server.URL)
	}

	for range 2 {
		select {
		case <-requestsStarted:
		case result := <-requestResults:
			require.NoError(t, result.err)
			t.Fatalf("request finished before shutdown began with status %d", result.status)
		case <-time.After(5 * time.Second):
			t.Fatal("both servers did not receive an active request")
		}
	}

	shutdownDone := make(chan struct{})
	go func() {
		shutdownHTTPServers(context.Background(), publicServer.Config, internalServer.Config)
		close(shutdownDone)
	}()
	for range 2 {
		select {
		case <-shutdownStarted:
		case <-time.After(5 * time.Second):
			release()
			<-shutdownDone
			t.Fatal("both servers must begin shutdown before active requests finish")
		}
	}
	release()
	for range 2 {
		select {
		case result := <-requestResults:
			require.NoError(t, result.err)
			assert.Equal(t, http.StatusNoContent, result.status)
		case <-time.After(5 * time.Second):
			t.Fatal("active requests did not drain")
		}
	}
	select {
	case <-shutdownDone:
	case <-time.After(5 * time.Second):
		t.Fatal("servers did not finish shutdown")
	}
}

func writeInternalTLSKeyPair(t *testing.T) (string, string) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	now := time.Now()
	certificate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "new-api.al-bei.cn"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"new-api.al-bei.cn"},
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, certificate, certificate, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	certFile := filepath.Join(t.TempDir(), "server.crt")
	keyFile := filepath.Join(t.TempDir(), "server.key")
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER}), 0o600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER}), 0o600))
	return certFile, keyFile
}
