package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/suapapa/si-gnal/internal/poem"
)

// AI wraps an OpenAI-compatible HTTP client for poem cleanup and TTS script generation.
type AI struct {
	client *openai.Client
	model  string
}

// NewAI builds an OpenAI-compatible chat client. baseURL may be empty to use
// the library default (https://api.openai.com/v1). Trailing slashes on baseURL are trimmed.
func NewAI(ctx context.Context, baseURL, apiKey, model string) (*AI, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context: %w", err)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey is empty")
	}
	if model == "" {
		return nil, fmt.Errorf("model is empty")
	}

	cfg := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		cfg.BaseURL = strings.TrimRight(baseURL, "/")
	}

	return &AI{client: openai.NewClientWithConfig(cfg), model: model}, nil
}

// Close releases any client resources (currently a no-op).
func (a *AI) Close() {}

// CleanupContent asks the model to strip metadata from p.Content and replaces p.Content.
func (a *AI) CleanupContent(ctx context.Context, p *poem.Poem) error {
	prompt := fmt.Sprintf(`주어진 시의 내용(content)에서 제목('%s')과 작가('%s') 정보, 그리고 불필요한 설명이 있다면 모두 제거하고 오직 '시의 본문'만 남겨줘.
수정된 시의 본문만 출력하고, 다른 설명이나 말은 절대 덧붙이지 마.

시 내용:
%s`, p.Title, p.Author, p.Content)

	content, err := a.generate(ctx, prompt)
	if err != nil {
		return err
	}

	p.Content = content
	return nil
}

// FillReadingScript sets p.ReadingScript to a TTS-friendly reading of p.Content.
func (a *AI) FillReadingScript(ctx context.Context, p *poem.Poem) error {
	prompt := fmt.Sprintf(`주어진 시 내용(content)을 TTS를 통해 읽으려고해.
원문을 존중하되 줄바꿈, 구두점을 수정하거나 추가해서 자연스럽게 읽을 수 있게 낭독용 대본으로 만들어줘.
설명은 제외하고 수정된 본문만 출력해.

시 내용:
%s`, p.Content)

	content, err := a.generate(ctx, prompt)
	if err != nil {
		return err
	}

	p.ReadingScript = content
	return nil
}

func (a *AI) generate(ctx context.Context, prompt string) (string, error) {
	resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: a.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("CreateChatCompletion: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in completion response")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}
