package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	projectIDHeader         = "X-AI-Project-Id"
	enterpriseIDHeader      = "X-AI-Enterprise-Id"
	featureHeader           = "X-AI-Feature"
	projectRequestIDHeader  = "X-Request-Id"
	platformRequestIDHeader = "X-Oneapi-Request-Id"
)

type bridgeConfig struct {
	listenAddr          string
	upstreamBaseURL     *url.URL
	projectID           string
	projectKey          string
	credentialVerifyURL string
	enterpriseTokens    map[string]string
}

func loadConfigFromEnv(getenv func(string) string) (bridgeConfig, error) {
	listenAddr := getenv("AI_BRIDGE_LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":3001"
	}
	baseURL := getenv("AI_BRIDGE_UPSTREAM_BASE_URL")
	projectID := getenv("AI_BRIDGE_PROJECT_ID")
	projectKey := getenv("AI_BRIDGE_PROJECT_KEY")
	verifyURL := getenv("AI_BRIDGE_CREDENTIAL_VERIFY_URL")
	if verifyURL != "" {
		parsed, err := url.Parse(verifyURL)
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return bridgeConfig{}, errors.New("AI_BRIDGE_CREDENTIAL_VERIFY_URL 必须是受信任的平台 HTTP(S) 地址")
		}
	} else if strings.TrimSpace(projectKey) == "" {
		return bridgeConfig{}, errors.New("未配置项目凭证校验地址或 POC 项目凭证")
	}
	for name, value := range map[string]string{
		"AI_BRIDGE_UPSTREAM_BASE_URL": baseURL,
		"AI_BRIDGE_PROJECT_ID":        projectID,
	} {
		if strings.TrimSpace(value) == "" {
			return bridgeConfig{}, fmt.Errorf("%s 未配置", name)
		}
	}

	upstream, err := url.Parse(baseURL)
	if err != nil || upstream.Host == "" || (upstream.Scheme != "http" && upstream.Scheme != "https") {
		return bridgeConfig{}, errors.New("AI_BRIDGE_UPSTREAM_BASE_URL 必须是合法的 HTTP(S) 地址")
	}
	tokens, err := parseEnterpriseTokens(getenv("AI_BRIDGE_ENTERPRISE_TOKENS"))
	if err != nil {
		return bridgeConfig{}, err
	}
	if !validIdentifier(projectID) {
		return bridgeConfig{}, errors.New("AI_BRIDGE_PROJECT_ID 格式无效")
	}
	return bridgeConfig{
		listenAddr:          listenAddr,
		upstreamBaseURL:     upstream,
		projectID:           projectID,
		projectKey:          projectKey,
		credentialVerifyURL: verifyURL,
		enterpriseTokens:    tokens,
	}, nil
}

func parseEnterpriseTokens(raw string) (map[string]string, error) {
	result := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 || !validIdentifier(strings.TrimSpace(parts[0])) || strings.TrimSpace(parts[1]) == "" {
			return nil, errors.New("AI_BRIDGE_ENTERPRISE_TOKENS 包含无效映射")
		}
		enterpriseID := strings.TrimSpace(parts[0])
		if _, exists := result[enterpriseID]; exists {
			return nil, errors.New("AI_BRIDGE_ENTERPRISE_TOKENS 存在重复企业标识")
		}
		result[enterpriseID] = strings.TrimSpace(parts[1])
	}
	if len(result) == 0 {
		return nil, errors.New("AI_BRIDGE_ENTERPRISE_TOKENS 未配置")
	}
	return result, nil
}

func validIdentifier(value string) bool {
	if len(value) == 0 || len(value) > 64 {
		return false
	}
	for index, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' || char == ':' {
			if index == 0 && !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
				return false
			}
			continue
		}
		return false
	}
	return true
}

func newBridgeHandler(config bridgeConfig) http.Handler {
	// 不跟随重定向，防止项目凭证被带往其他地址；每次请求实时校验，不缓存放行结果。
	verifyClient := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	mux := http.NewServeMux()
	mux.HandleFunc("/health/ready", func(response http.ResponseWriter, request *http.Request) {
		writeJSON(response, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1" && !strings.HasPrefix(request.URL.Path, "/v1/") {
			writeJSON(response, http.StatusNotFound, map[string]string{"error": "仅允许转发 /v1/* 接口"})
			return
		}
		if request.Header.Get(projectIDHeader) != config.projectID {
			writeJSON(response, http.StatusForbidden, map[string]string{"error": "项目凭证无效"})
			return
		}
		projectKey := bearerToken(request.Header.Get("Authorization"))
		if projectKey == "" || len(projectKey) > 256 {
			writeJSON(response, http.StatusUnauthorized, map[string]string{"error": "项目凭证无效"})
			return
		}
		if config.credentialVerifyURL != "" {
			ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
			defer cancel()
			verifyRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, config.credentialVerifyURL, nil)
			if err != nil {
				writeJSON(response, http.StatusServiceUnavailable, map[string]string{"error": "项目凭证校验暂时不可用"})
				return
			}
			verifyRequest.Header.Set("Authorization", "Bearer "+projectKey)
			verifyRequest.Header.Set(projectIDHeader, config.projectID)
			verified, err := verifyClient.Do(verifyRequest)
			if err != nil {
				writeJSON(response, http.StatusServiceUnavailable, map[string]string{"error": "项目凭证校验暂时不可用"})
				return
			}
			verified.Body.Close()
			if verified.StatusCode != http.StatusNoContent {
				status := http.StatusServiceUnavailable
				if verified.StatusCode == http.StatusUnauthorized {
					status = http.StatusUnauthorized
				}
				writeJSON(response, status, map[string]string{"error": "项目凭证校验未通过"})
				return
			}
		} else if subtle.ConstantTimeCompare([]byte(projectKey), []byte(config.projectKey)) != 1 {
			writeJSON(response, http.StatusUnauthorized, map[string]string{"error": "项目凭证无效"})
			return
		}
		enterpriseID := strings.TrimSpace(request.Header.Get(enterpriseIDHeader))
		if !validIdentifier(enterpriseID) {
			writeJSON(response, http.StatusBadRequest, map[string]string{"error": "企业标识无效"})
			return
		}
		upstreamToken, exists := config.enterpriseTokens[enterpriseID]
		if !exists {
			writeJSON(response, http.StatusForbidden, map[string]string{"error": "企业尚未开通 AI 额度"})
			return
		}
		feature := strings.TrimSpace(request.Header.Get(featureHeader))
		if !validIdentifier(feature) {
			writeJSON(response, http.StatusBadRequest, map[string]string{"error": "功能标识无效"})
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(config.upstreamBaseURL)
		director := proxy.Director
		directorRequestID := request.Header.Get(projectRequestIDHeader)
		proxy.Director = func(upstreamRequest *http.Request) {
			director(upstreamRequest)
			// 中转站只把映射后的 New API Key 发给上游，项目凭证不会透传。
			upstreamRequest.Header.Set("Authorization", "Bearer "+upstreamToken)
			upstreamRequest.Header.Set(projectIDHeader, config.projectID)
			upstreamRequest.Header.Set(enterpriseIDHeader, enterpriseID)
			upstreamRequest.Header.Set(featureHeader, feature)
			if directorRequestID != "" {
				// 业务项目使用通用请求号，New API 使用自己的请求号请求头，需在 bridge 处适配。
				upstreamRequest.Header.Set(platformRequestIDHeader, directorRequestID)
			}
		}
		proxy.ErrorHandler = func(errorResponse http.ResponseWriter, _ *http.Request, _ error) {
			writeJSON(errorResponse, http.StatusBadGateway, map[string]string{"error": "AI 平台上游暂时不可用"})
		}
		proxy.ServeHTTP(response, request)
	})
	return withRequestTimeout(mux)
}

func withRequestTimeout(handler http.Handler) http.Handler {
	return http.TimeoutHandler(handler, 5*time.Minute, `{"error":"请求超时"}`)
}

func bearerToken(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < len("Bearer ") || !strings.EqualFold(value[:len("Bearer ")], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(value[len("Bearer "):])
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func main() {
	config, err := loadConfigFromEnv(os.Getenv)
	if err != nil {
		log.Fatal("AI 项目中转层配置无效：", err)
	}
	server := &http.Server{
		Addr:              config.listenAddr,
		Handler:           newBridgeHandler(config),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("AI 项目中转层已启动，项目=%s，上游=%s", config.projectID, config.upstreamBaseURL.Redacted())
	log.Fatal(server.ListenAndServe())
}
