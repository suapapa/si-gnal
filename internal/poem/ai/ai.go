package ai

import (
	"context"
	"fmt"
	"strings"
	"unicode"

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

한자 병기는 TTS가 같은 단어를 두 번 읽지 않도록 반드시 아래 규칙대로 처리해줘 (반각 괄호 ()와 전각 괄호 （） 모두 해당):
- 한글 단어 뒤에 괄호로 한자를 병기한 경우, 괄호와 그 안의 한자를 모두 제거하고 한글 단어만 한 번 남겨줘. 예: "산보로(散步路)"는 "산보로"로, "고향(故鄕)"은 "고향"으로 만들어줘.
- 반대로 한자 뒤에 괄호로 한글 독음을 병기한 경우, 한자와 괄호 표기 전체를 한글 독음 한 번으로 바꿔줘. 예: "散步路(산보로)"는 "산보로"로 만들어줘.
- 괄호 안 한글 독음이 앞 단어와 같거나 앞 한자의 독음과 같은 경우에도, 어느 한쪽만 남기고 같은 단어를 두 번 연속으로 쓰지 마.
- 괄호 없이 본문에 남은 한자도 한글 독음으로 풀어서 한 번만 써줘. 낭독용 대본에는 한자를 남기지 마.
- 단, 괄호 안 내용이 한자 병기나 독음이 아니라 시의 실제 내용·보충 설명인 경우에는 이 규칙을 적용해 임의로 삭제하지 말고 자연스럽게 읽을 수 있게 다듬어줘.

시 내용:
%s`, p.Content)

	content, err := a.generate(ctx, prompt)
	if err != nil {
		return err
	}

	// 모델이 위 규칙을 지키지 않더라도 TTS가 병기된 한자를 다시 읽어
	// 같은 단어를 두 번 낭독하지 않도록 결정적으로 한 번 더 정리한다.
	p.ReadingScript = normalizeReadingScript(content)
	return nil
}

// normalizeReadingScript removes Hanja annotations written in parentheses
// after a Hangul word (e.g. "산보로(散步路)" → "산보로"), and rewrites the
// reverse form (e.g. "散步路(산보로)" → "산보로"), so TTS does not read the
// same word twice. Parenthesised text that is neither a Hanja annotation
// nor a duplicated reading is left untouched.
func normalizeReadingScript(s string) string {
	runes := []rune(s)
	var out []rune
	for i := 0; i < len(runes); i++ {
		if runes[i] != '(' && runes[i] != '（' {
			out = append(out, runes[i])
			continue
		}
		closeIdx := -1
		for j := i + 1; j < len(runes); j++ {
			if runes[j] == ')' || runes[j] == '）' {
				closeIdx = j
				break
			}
		}
		if closeIdx < 0 {
			out = append(out, runes[i])
			continue
		}
		inside := strings.TrimSpace(string(runes[i+1 : closeIdx]))
		switch {
		case isAllHan(inside):
			// 산보로(散步路) → 산보로: 병기된 한자만 버린다.
			out = trimTrailingSpaces(out)
			i = closeIdx
		case isAllHangul(inside) && hasTrailingHan(out):
			// 散步路(산보로) → 산보로: 앞 한자를 한글 독음으로 교체한다.
			out = append(trimTrailingHan(out), []rune(inside)...)
			i = closeIdx
		case isAllHangul(inside) && trailingHangulWord(out) == inside:
			// 산보로(산보로) → 산보로: 같은 단어의 반복을 버린다.
			out = trimTrailingSpaces(out)
			i = closeIdx
		default:
			out = append(out, runes[i])
		}
	}
	return string(out)
}

func isAllHan(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || r == ' ' || r == '·' || r == ',' || r == '、' {
			continue
		}
		return false
	}
	return true
}

func isAllHangul(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.Is(unicode.Hangul, r) {
			return false
		}
	}
	return true
}

func hasTrailingHan(out []rune) bool {
	out = trimTrailingSpaces(out)
	return len(out) > 0 && unicode.Is(unicode.Han, out[len(out)-1])
}

func trimTrailingSpaces(out []rune) []rune {
	for len(out) > 0 && (out[len(out)-1] == ' ' || out[len(out)-1] == '\t') {
		out = out[:len(out)-1]
	}
	return out
}

func trimTrailingHan(out []rune) []rune {
	out = trimTrailingSpaces(out)
	for len(out) > 0 && unicode.Is(unicode.Han, out[len(out)-1]) {
		out = out[:len(out)-1]
	}
	return out
}

func trailingHangulWord(out []rune) string {
	out = trimTrailingSpaces(out)
	start := len(out)
	for start > 0 && unicode.Is(unicode.Hangul, out[start-1]) {
		start--
	}
	return string(out[start:])
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
