package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/goccy/go-yaml"
	"github.com/suapapa/si-gnal/internal/poem"
	"github.com/suapapa/si-gnal/internal/poem/ai"
)

type outputFormat string

const (
	formatYAML outputFormat = "yaml"
	formatJSON outputFormat = "json"
	formatTXT  outputFormat = "txt"
)

func parseOutputFormat(s string) (outputFormat, error) {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "yaml":
		return formatYAML, nil
	case "json":
		return formatJSON, nil
	case "txt":
		return formatTXT, nil
	default:
		return "", fmt.Errorf("알 수 없는 출력 형식: %q (txt|yaml|json)", s)
	}
}

func main() {
	outFmtStr := flag.String("f", "yaml", "txt|yaml|json")
	outFile := flag.String("o", "", "output file")
	flag.Parse()

	format, err := parseOutputFormat(*outFmtStr)
	if err != nil {
		log.Fatalf("%v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, format, *outFile); err != nil {
		log.Fatalf("%v", err)
	}
}

func run(ctx context.Context, outFmt outputFormat, outFile string) error {
	log.Println("게시판 정보를 확인 중입니다...")

	lastPage, err := poem.GetLastPage(ctx)
	if err != nil {
		return fmt.Errorf("마지막 페이지: %w", err)
	}
	log.Printf("전체 %d개 페이지 중 무작위 선택 중...", lastPage)

	randomPage := rand.Intn(lastPage) + 1
	wrIDs, err := poem.GetPoemLinks(ctx, randomPage)
	if err != nil {
		return fmt.Errorf("시 링크: %w", err)
	}

	if len(wrIDs) == 0 {
		return fmt.Errorf("시 목록을 가져오지 못했습니다 (page=%d)", randomPage)
	}

	randomWrID := wrIDs[rand.Intn(len(wrIDs))]
	p, err := poem.GetPoemDetail(ctx, randomWrID)
	if err != nil {
		return fmt.Errorf("시 상세: %w", err)
	}

	if p == nil {
		return fmt.Errorf("시 내용을 불러오지 못했습니다")
	}

	if err := maybeApplyAI(ctx, p); err != nil {
		return err
	}

	if err := writeStdout(outFmt, p); err != nil {
		return err
	}

	if outFile == "" {
		return nil
	}

	b, err := marshalPoem(outFmt, p)
	if err != nil {
		return fmt.Errorf("결과 인코딩: %w", err)
	}

	if err := os.WriteFile(outFile, b, 0o644); err != nil {
		return fmt.Errorf("파일 저장: %w", err)
	}
	log.Printf("결과가 %s에 저장되었습니다.", outFile)
	return nil
}

func maybeApplyAI(ctx context.Context, p *poem.Poem) error {
	baseURL := os.Getenv("SIGNAL_OPENAI_BASE_URL")
	if baseURL == "" {
		return nil
	}
	apiKey := os.Getenv("SIGNAL_OPENAI_API_KEY")
	model := os.Getenv("SIGNAL_OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	aiFix, err := ai.NewAI(ctx, baseURL, apiKey, model)
	if err != nil {
		return fmt.Errorf("AI 클라이언트: %w", err)
	}
	defer aiFix.Close()

	log.Println("AI 교정 중...")
	if err := aiFix.CleanupContent(ctx, p); err != nil {
		return fmt.Errorf("AI 교정: %w", err)
	}
	log.Println("AI 낭송 대본 생성 중...")
	if err := aiFix.FillReadingScript(ctx, p); err != nil {
		return fmt.Errorf("낭송 대본: %w", err)
	}
	return nil
}

func marshalPoem(f outputFormat, p *poem.Poem) ([]byte, error) {
	switch f {
	case formatJSON:
		b, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("json: %w", err)
		}
		return b, nil
	case formatYAML:
		b, err := yaml.Marshal(p)
		if err != nil {
			return nil, fmt.Errorf("yaml: %w", err)
		}
		return b, nil
	case formatTXT:
		return []byte(formatPoemTextFile(p)), nil
	default:
		return nil, fmt.Errorf("unknown format %q", f)
	}
}

func writeStdout(f outputFormat, p *poem.Poem) error {
	switch f {
	case formatTXT:
		fmt.Print(formatPoemTextTTY(p))
		return nil
	case formatJSON, formatYAML:
		b, err := marshalPoem(f, p)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	default:
		return fmt.Errorf("unknown format %q", f)
	}
}

func formatPoemTextFile(p *poem.Poem) string {
	sep := strings.Repeat("-", 50)
	txt := fmt.Sprintf("📜 %s\n👤 저자: %s\n%s\n%s\n\n%s\n🔗 출처: %s\n",
		p.Title, p.Author, sep, p.Content, sep, p.URL)
	if p.ReadingScript != "" {
		txt += fmt.Sprintf("\n📣 낭송 대본:\n%s\n", p.ReadingScript)
	}
	return txt
}

func formatPoemTextTTY(p *poem.Poem) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(strings.Repeat("=", 50))
	b.WriteByte('\n')
	_, _ = fmt.Fprintf(&b, "📜 %s\n", p.Title)
	_, _ = fmt.Fprintf(&b, "👤 저자: %s\n", p.Author)
	b.WriteString(strings.Repeat("-", 50))
	b.WriteString("\n\n")
	b.WriteString(p.Content)
	b.WriteString("\n\n")
	if p.ReadingScript != "" {
		b.WriteString(strings.Repeat("-", 50))
		b.WriteString("\n📣 낭송 대본:\n\n")
		b.WriteString(p.ReadingScript)
		b.WriteString("\n\n")
	}
	b.WriteString(strings.Repeat("=", 50))
	_, _ = fmt.Fprintf(&b, "\n🔗 출처: %s\n", p.URL)
	return b.String()
}
