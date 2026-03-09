package poem

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// AIFix uses Gemini to remove title, author, and unnecessary descriptions from the content.
func (p *Poem) AIFix(ctx context.Context) error {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY is not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return fmt.Errorf("failed to create genai client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash-lite")

	prompt := fmt.Sprintf(`주어진 시의 내용(content)에서 제목('%s')과 작가('%s') 정보, 그리고 불필요한 설명이 있다면 모두 제거하고 오직 '시의 본문'만 남겨줘.
수정된 시의 본문만 출력하고, 다른 설명이나 말은 절대 덧붙이지 마.

시 내용:
%s`, p.Title, p.Author, p.Content)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return fmt.Errorf("GenerateContent error: %w", err)
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		if part, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
			p.Content = strings.TrimSpace(string(part))
		}
	}
	return nil
}
