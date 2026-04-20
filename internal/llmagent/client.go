package llmagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client interface {
	Decide(ctx context.Context, messages []Message) (Decision, error)
}

type OpenAICompatibleClient struct {
	baseURL     string
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	httpClient  *http.Client
}

func NewOpenAICompatibleClient(baseURL, apiKey, model string, timeout time.Duration, maxTokens int, temperature float64) *OpenAICompatibleClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &OpenAICompatibleClient{
		baseURL:     strings.TrimRight(baseURL, "/"),
		apiKey:      apiKey,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
		httpClient:  &http.Client{Timeout: timeout},
	}
}

func (c *OpenAICompatibleClient) Decide(ctx context.Context, messages []Message) (Decision, error) {
	if c.baseURL == "" {
		return Decision{}, fmt.Errorf("llm base URL is required")
	}
	if c.apiKey == "" {
		return Decision{}, fmt.Errorf("llm API key is required")
	}
	if c.model == "" {
		return Decision{}, fmt.Errorf("llm model is required")
	}

	body := map[string]interface{}{
		"model":       c.model,
		"messages":    messages,
		"temperature": c.temperature,
		"max_tokens":  c.maxTokens,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return Decision{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return Decision{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return Decision{}, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return Decision{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Decision{}, fmt.Errorf("llm request failed: status=%d body=%s", res.StatusCode, string(resBody))
	}

	var decoded struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(resBody, &decoded); err != nil {
		return Decision{}, err
	}
	if len(decoded.Choices) == 0 {
		return Decision{}, fmt.Errorf("llm response contained no choices")
	}
	return ParseDecision(decoded.Choices[0].Message.Content)
}

func ParseDecision(content string) (Decision, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var decision Decision
	if err := json.Unmarshal([]byte(content), &decision); err != nil {
		return Decision{}, fmt.Errorf("parse llm decision: %w", err)
	}
	switch decision.Action {
	case ActionTool:
		if strings.TrimSpace(decision.Tool) == "" {
			return Decision{}, fmt.Errorf("tool decision requires tool name")
		}
		if decision.Args == nil {
			decision.Args = map[string]interface{}{}
		}
	case ActionFinal:
		if strings.TrimSpace(decision.FinalAnswer) == "" {
			return Decision{}, fmt.Errorf("final decision requires final_answer")
		}
	default:
		return Decision{}, fmt.Errorf("unsupported decision action %q", decision.Action)
	}
	return decision, nil
}
