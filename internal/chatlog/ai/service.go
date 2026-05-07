package ai

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/sjzar/chatlog/internal/chatlog/conf"
)

// ProviderType 已支持的提供商类型常量。
const (
	TypeOpenAI           = "openai"
	TypeOpenAICompatible = "openai-compatible"
	TypeAnthropic        = "anthropic"
	TypeGoogle           = "google"
)

// Service 负责 AI 提供商的连通性测试与模型查询。
// 实际生成调用通过 goai SDK 在更上层实现，本服务只提供轻量探针。
type Service struct {
	client *resty.Client
}

func New() *Service {
	c := resty.New()
	c.SetTimeout(15 * time.Second)
	c.SetHeader("Accept", "application/json")
	return &Service{client: c}
}

// TestResult 是连通性测试的结果摘要。
type TestResult struct {
	OK       bool   `json:"ok"`
	Latency  int64  `json:"latency_ms"`
	Endpoint string `json:"endpoint"`
	Status   int    `json:"status,omitempty"`
	Message  string `json:"message,omitempty"`
}

// TestProvider 校验 API Key 和 BaseURL 是否可达。
// 对 OpenAI / OpenAI 兼容类型使用 GET {base}/v1/models；
// Anthropic 使用 GET https://api.anthropic.com/v1/models（带 anthropic-version 头）。
func (s *Service) TestProvider(ctx context.Context, p *conf.AIProvider) TestResult {
	if p == nil {
		return TestResult{Message: "provider 为空"}
	}
	if strings.TrimSpace(p.APIKey) == "" {
		return TestResult{Message: "API Key 为空"}
	}

	endpoint, headers, err := buildModelsRequest(p)
	if err != nil {
		return TestResult{Message: err.Error()}
	}

	req := s.client.R().SetContext(ctx).SetHeaders(headers)
	start := time.Now()
	resp, err := req.Get(endpoint)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return TestResult{Endpoint: endpoint, Latency: latency, Message: err.Error()}
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return TestResult{
			Endpoint: endpoint,
			Latency:  latency,
			Status:   resp.StatusCode(),
			Message:  trimBody(resp.String()),
		}
	}
	return TestResult{OK: true, Endpoint: endpoint, Latency: latency, Status: resp.StatusCode()}
}

// ListModels 拉取提供商可用模型 ID 列表（OpenAI 兼容协议）。
func (s *Service) ListModels(ctx context.Context, p *conf.AIProvider) ([]string, error) {
	if p == nil {
		return nil, errors.New("provider 为空")
	}
	if strings.TrimSpace(p.APIKey) == "" {
		return nil, errors.New("API Key 为空")
	}

	endpoint, headers, err := buildModelsRequest(p)
	if err != nil {
		return nil, err
	}

	type modelItem struct {
		ID string `json:"id"`
	}
	type modelsResp struct {
		Data   []modelItem `json:"data"`
		Models []modelItem `json:"models"`
	}
	var out modelsResp

	resp, err := s.client.R().
		SetContext(ctx).
		SetHeaders(headers).
		SetResult(&out).
		Get(endpoint)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode(), trimBody(resp.String()))
	}

	items := out.Data
	if len(items) == 0 {
		items = out.Models
	}
	ids := make([]string, 0, len(items))
	for _, m := range items {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func buildModelsRequest(p *conf.AIProvider) (string, map[string]string, error) {
	headers := map[string]string{}
	switch strings.ToLower(p.Type) {
	case TypeOpenAI, TypeOpenAICompatible, "":
		base := strings.TrimRight(p.BaseURL, "/")
		if base == "" {
			base = "https://api.openai.com"
		}
		headers["Authorization"] = "Bearer " + p.APIKey
		return base + "/v1/models", headers, nil
	case TypeAnthropic:
		base := strings.TrimRight(p.BaseURL, "/")
		if base == "" {
			base = "https://api.anthropic.com"
		}
		headers["x-api-key"] = p.APIKey
		headers["anthropic-version"] = "2023-06-01"
		return base + "/v1/models", headers, nil
	case TypeGoogle:
		base := strings.TrimRight(p.BaseURL, "/")
		if base == "" {
			base = "https://generativelanguage.googleapis.com"
		}
		return base + "/v1beta/models?key=" + p.APIKey, map[string]string{}, nil
	default:
		return "", nil, fmt.Errorf("不支持的提供商类型：%s", p.Type)
	}
}

func trimBody(body string) string {
	body = strings.TrimSpace(body)
	if len(body) > 280 {
		return body[:280] + "…"
	}
	return body
}
