package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/suapapa/si-gnal/internal/poem"
	"github.com/suapapa/si-gnal/internal/poem/ai"
)

func main() {

	var outFmt string
	var outFile string
	flag.StringVar(&outFmt, "f", "yaml", "txt|yaml|json")
	flag.StringVar(&outFile, "o", "", "output file")
	flag.Parse()

	log.Println("게시판 정보를 확인 중입니다...")

	lastPage, err := poem.GetLastPage()
	if err != nil {
		log.Printf("오류: %v\n", err)
		return
	}
	log.Printf("전체 %d개 페이지 중 무작위 선택 중...", lastPage)

	randomPage := rand.Intn(lastPage) + 1
	wrIDs, err := poem.GetPoemLinks(randomPage)
	if err != nil {
		log.Printf("오류: %v", err)
		return
	}

	if len(wrIDs) == 0 {
		log.Println("시 목록을 가져오지 못했습니다.")
		return
	}

	randomWrID := wrIDs[rand.Intn(len(wrIDs))]
	p, err := poem.GetPoemDetail(randomWrID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "오류: %v\n", err)
		return
	}

	if p == nil {
		log.Println("시 내용을 불러오지 못했습니다.")
		os.Exit(1)
	}

	if os.Getenv("GEMINI_API_KEY") != "" {
		aiFix, err := ai.NewAI(context.Background())
		if err != nil {
			log.Printf("오류: %v\n", err)
			return
		}
		defer aiFix.Close()
		log.Println("AI 교정 중...")
		aiFix.CleanupContent(context.Background(), p)
		log.Println("AI 낭송 대본 생성 중...")
		aiFix.FillReadingScript(context.Background(), p)
	}

	switch outFmt {
	case "txt":
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Printf("📜 %s\n", p.Title)
		fmt.Printf("👤 저자: %s\n", p.Author)
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("\n%s\n\n", p.Content)
		if p.ReadingScript != "" {
			fmt.Println(strings.Repeat("-", 50))
			fmt.Println("📣 낭송 대본:")
			fmt.Printf("\n%s\n\n", p.ReadingScript)
		}
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("🔗 출처: %s\n", p.URL)
	case "json":
		b, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			fmt.Printf("JSON 변환 오류: %v\n", err)
			return
		}
		fmt.Println(string(b))
	case "yaml":
		b, err := yaml.Marshal(p)
		if err != nil {
			fmt.Printf("YAML 변환 오류: %v\n", err)
			return
		}
		fmt.Println(string(b))
	default:
		fmt.Printf("알 수 없는 출력 형식: %s\n", outFmt)
	}

	if outFile != "" {
		var b []byte
		var err error
		switch outFmt {
		case "json":
			b, err = json.MarshalIndent(p, "", "  ")
		case "yaml":
			b, err = yaml.Marshal(p)
		case "txt":
			txt := fmt.Sprintf("📜 %s\n👤 저자: %s\n%s\n\n%s\n%s\n🔗 출처: %s\n",
				p.Title, p.Author, strings.Repeat("-", 50), p.Content, strings.Repeat("-", 50), p.URL)
			if p.ReadingScript != "" {
				txt += fmt.Sprintf("\n📣 낭송 대본:\n%s\n", p.ReadingScript)
			}
			b = []byte(txt)
		default:
			log.Fatalf("알 수 없는 출력 형식: %s", outFmt)
		}

		if err != nil {
			log.Fatalf("결과 저장 준비 중 오류: %v", err)
		}

		err = os.WriteFile(outFile, b, 0644)
		if err != nil {
			log.Fatalf("파일 저장 오류: %v", err)
		}
		log.Printf("결과가 %s에 저장되었습니다.", outFile)
	}

}
