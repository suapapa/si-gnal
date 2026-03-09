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
	"github.com/suapapa/signal/internal/poem"
)

func main() {

	var outFmt string
	flag.StringVar(&outFmt, "f", "yaml", "txt|yaml|json")
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
		if err := p.AIFix(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "AI 교정 중 오류 발생: %v\n", err)
		}
	}

	switch outFmt {
	case "txt":
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Printf("📜 %s\n", p.Title)
		fmt.Printf("👤 저자: %s\n", p.Author)
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("\n%s\n\n", p.Content)
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
}
