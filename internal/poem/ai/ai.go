package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/suapapa/si-gnal/internal/poem"
	"google.golang.org/api/option"
)

type AI struct {
	client *genai.Client
}

func NewAI(ctx context.Context) (*AI, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	// models := client.ListModels(ctx)
	// for m, err := models.Next(); err == nil; m, err = models.Next() {
	// 	log.Println(m.Name)
	// }

	return &AI{client: client}, nil
}

func (a *AI) Close() {
	a.client.Close()
}

func (a *AI) CleanupContent(ctx context.Context, p *poem.Poem) error {
	prompt := fmt.Sprintf(`주어진 시의 내용(content)에서 제목('%s')과 작가('%s') 정보, 그리고 불필요한 설명이 있다면 모두 제거하고 오직 '시의 본문'만 남겨줘.
수정된 시의 본문만 출력하고, 다른 설명이나 말은 절대 덧붙이지 마.

시 내용:
%s`, p.Title, p.Author, p.Content)

	model := "gemma-3-12b-it" //"gemini-2.5-flash-lite"
	content, err := a.generate(ctx, model, prompt)
	if err != nil {
		return err
	}

	p.Content = content
	return nil
}

func (a *AI) FillReadingScript(ctx context.Context, p *poem.Poem) error {
	prompt := fmt.Sprintf(`주어진 시 본문을 TTS를 통해 읽으려해. 줄바꿈, 구두점을 수정하거나 추가해서 자연스럽게 읽을 수 있게 낭독용 대본으로 만들어줘.
원문을 존중해서 줄 순서를 바꾸지는 마.
설명은 제외하고 수정된 본문만 출력해.
---
%s`, p.Content)

	model := "gemma-3-27b-it" //"gemini-2.5-flash"
	content, err := a.generate(ctx, model, prompt)
	if err != nil {
		return err
	}

	p.ReadingScript = content
	return nil
}

func (a *AI) generate(ctx context.Context, modelName string, prompt string) (string, error) {
	model := a.client.GenerativeModel(modelName)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("GenerateContent error: %w", err)
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		if part, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
			return strings.TrimSpace(string(part)), nil
		}
	}
	return "", fmt.Errorf("no content generated")
}
