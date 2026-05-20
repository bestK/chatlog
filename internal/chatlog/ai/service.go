package ai

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sashabaranov/go-openai"

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

// SummaryResult 是 AI 生成的总结结果
type SummaryResult struct {
	Summary string `json:"summary"`
	Model   string `json:"model"`
	Tokens  int    `json:"tokens,omitempty"`
}

// GenerateSummary 使用指定提供商和模型生成聊天记录总结
func (s *Service) GenerateSummary(ctx context.Context, p *conf.AIProvider, messages []string, prompt string) (*SummaryResult, error) {
	if p == nil || p.Disabled {
		return nil, errors.New("provider 为空或已禁用")
	}
	if strings.TrimSpace(p.APIKey) == "" {
		return nil, errors.New("API Key 为空")
	}

	// 构建提示词
	chatContent := strings.Join(messages, "\n\n---\n\n")
	if strings.TrimSpace(prompt) == "" {
		prompt = "请总结以下聊天记录的主要内容，包括：\n1. 聊天主题\n2. 关键讨论点\n3. 重要结论或待办事项\n\n聊天记录："
	}
	fullPrompt := prompt + "\n\n" + chatContent

	// 根据提供商类型调用不同 API
	switch strings.ToLower(p.Type) {
	case TypeOpenAI, TypeOpenAICompatible, "":
		return s.generateOpenAI(ctx, p, fullPrompt)
	case TypeAnthropic:
		return s.generateAnthropic(ctx, p, fullPrompt)
	case TypeGoogle:
		return s.generateGoogle(ctx, p, fullPrompt)
	default:
		return nil, fmt.Errorf("不支持的提供商类型：%s", p.Type)
	}
}

// GenerateSummaryStream 流式生成聊天记录总结
func (s *Service) GenerateSummaryStream(ctx context.Context, p *conf.AIProvider, messages []string, prompt string) (<-chan string, error) {
	if p == nil || p.Disabled {
		return nil, errors.New("provider 为空或已禁用")
	}
	if strings.TrimSpace(p.APIKey) == "" {
		return nil, errors.New("API Key 为空")
	}

	// 构建提示词
	chatContent := strings.Join(messages, "\n\n---\n\n")
	if strings.TrimSpace(prompt) == "" {
		prompt = "请总结以下聊天记录的主要内容，包括：\n1. 聊天主题\n2. 关键讨论点\n3. 重要结论或待办事项\n\n聊天记录："
	}
	fullPrompt := prompt + "\n\n" + chatContent

	// 根据提供商类型调用不同 API
	switch strings.ToLower(p.Type) {
	case TypeOpenAI, TypeOpenAICompatible, "":
		return s.generateOpenAIStream(ctx, p, fullPrompt)
	case TypeAnthropic:
		return s.generateAnthropicStream(ctx, p, fullPrompt)
	case TypeGoogle:
		return s.generateGoogleStream(ctx, p, fullPrompt)
	default:
		return nil, fmt.Errorf("不支持的提供商类型：%s", p.Type)
	}
}

func (s *Service) generateOpenAI(ctx context.Context, p *conf.AIProvider, prompt string) (*SummaryResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com"
	}
	model := p.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type requestBody struct {
		Model    string    `json:"model"`
		Messages []message `json:"messages"`
	}
	type responseBody struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	req := requestBody{
		Model: model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}
	var resp responseBody

	r, err := s.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+p.APIKey).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&resp).
		Post(base + "/v1/chat/completions")

	if err != nil {
		return nil, err
	}
	if resp.Error != nil && resp.Error.Message != "" {
		return nil, errors.New(resp.Error.Message)
	}
	if r.StatusCode() < 200 || r.StatusCode() >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", r.StatusCode(), trimBody(r.String()))
	}
	if len(resp.Choices) == 0 {
		return nil, errors.New("返回结果为空")
	}

	return &SummaryResult{
		Summary: strings.TrimSpace(resp.Choices[0].Message.Content),
		Model:   model,
		Tokens:  resp.Usage.TotalTokens,
	}, nil
}

func (s *Service) generateAnthropic(ctx context.Context, p *conf.AIProvider, prompt string) (*SummaryResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	if base == "" {
		base = "https://api.anthropic.com"
	}
	model := p.Model
	if model == "" {
		model = "claude-3-5-sonnet-latest"
	}

	type requestBody struct {
		Model     string `json:"model"`
		MaxTokens int    `json:"max_tokens"`
		Messages  []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	type responseBody struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	req := requestBody{
		Model:     model,
		MaxTokens: 4096,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "user", Content: prompt},
		},
	}
	var resp responseBody

	r, err := s.client.R().
		SetContext(ctx).
		SetHeader("x-api-key", p.APIKey).
		SetHeader("anthropic-version", "2023-06-01").
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&resp).
		Post(base + "/v1/messages")

	if err != nil {
		return nil, err
	}
	if resp.Error != nil && resp.Error.Message != "" {
		return nil, errors.New(resp.Error.Message)
	}
	if r.StatusCode() < 200 || r.StatusCode() >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", r.StatusCode(), trimBody(r.String()))
	}

	summary := ""
	for _, c := range resp.Content {
		if c.Type == "text" {
			summary += c.Text
		}
	}

	return &SummaryResult{
		Summary: strings.TrimSpace(summary),
		Model:   model,
		Tokens:  resp.Usage.InputTokens + resp.Usage.OutputTokens,
	}, nil
}

func (s *Service) generateGoogle(ctx context.Context, p *conf.AIProvider, prompt string) (*SummaryResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	if base == "" {
		base = "https://generativelanguage.googleapis.com"
	}
	model := p.Model
	if model == "" {
		model = "gemini-1.5-flash"
	}

	type requestBody struct {
		Contents []struct {
			Role  string `json:"role"`
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"contents"`
	}
	type responseBody struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	req := requestBody{
		Contents: []struct {
			Role  string `json:"role"`
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		}{
			{
				Role: "user",
				Parts: []struct {
					Text string `json:"text"`
				}{{Text: prompt}},
			},
		},
	}
	var resp responseBody

	r, err := s.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&resp).
		Post(base + "/v1beta/models/" + model + ":generateContent?key=" + p.APIKey)

	if err != nil {
		return nil, err
	}
	if resp.Error != nil && resp.Error.Message != "" {
		return nil, errors.New(resp.Error.Message)
	}
	if r.StatusCode() < 200 || r.StatusCode() >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", r.StatusCode(), trimBody(r.String()))
	}
	if len(resp.Candidates) == 0 {
		return nil, errors.New("返回结果为空")
	}

	summary := ""
	for _, p := range resp.Candidates[0].Content.Parts {
		summary += p.Text
	}

	return &SummaryResult{
		Summary: strings.TrimSpace(summary),
		Model:   model,
		Tokens:  0,
	}, nil
}

func trimBody(body string) string {
	body = strings.TrimSpace(body)
	if len(body) > 280 {
		return body[:280] + "…"
	}
	return body
}

// generateOpenAIStream OpenAI 流式生成
func (s *Service) generateOpenAIStream(ctx context.Context, p *conf.AIProvider, prompt string) (<-chan string, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com"
	}
	model := p.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	config := openai.DefaultConfig(p.APIKey)
	config.BaseURL = base + "/v1"
	client := openai.NewClientWithConfig(config)

	stream, err := client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Stream: true,
	})
	if err != nil {
		return nil, err
	}

	ch := make(chan string, 10)
	go func() {
		defer close(ch)
		for {
			resp, err := stream.Recv()
			if err != nil {
				break
			}
			if len(resp.Choices) > 0 && resp.Choices[0].Delta.Content != "" {
				select {
				case ch <- resp.Choices[0].Delta.Content:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// generateAnthropicStream Anthropic 流式生成（暂不支持）
func (s *Service) generateAnthropicStream(ctx context.Context, p *conf.AIProvider, prompt string) (<-chan string, error) {
	return nil, fmt.Errorf("Anthropic 流式生成暂不支持")
}

// generateGoogleStream Google 流式生成（暂不支持）
func (s *Service) generateGoogleStream(ctx context.Context, p *conf.AIProvider, prompt string) (<-chan string, error) {
	return nil, fmt.Errorf("Google 流式生成暂不支持")
}
