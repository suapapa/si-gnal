package poem

import (
	"fmt"
	stdhtml "html"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

const baseURL = "https://www.poemlove.co.kr/bbs/board.php?bo_table=tb01"

func setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Referer", "https://www.poemlove.co.kr/")
}

func GetLastPage() (int, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return 0, fmt.Errorf("마지막 페이지 확인 중 오류 발생: %w", err)
	}
	setHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("마지막 페이지 확인 중 오류 발생: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("마지막 페이지 확인 중 오류 발생: 상태 코드 %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("마지막 페이지 확인 중 오류 발생: %w", err)
	}

	lastPageIcon := doc.Find("i.fa-angle-double-right").First()
	if lastPageIcon.Length() > 0 {
		parent := lastPageIcon.Parent()
		if parent.Is("a") {
			if href, exists := parent.Attr("href"); exists {
				if strings.Contains(href, "page=") {
					parts := strings.Split(href, "page=")
					if len(parts) > 1 {
						pageStr := strings.Split(parts[1], "&")[0]
						if pageNum, err := strconv.Atoi(pageStr); err == nil {
							return pageNum, nil
						}
					}
				}
			}
		}
	}

	var maxPage int
	re := regexp.MustCompile(`page=(\d+)`)
	doc.Find("a[href*=\"page=\"]").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			match := re.FindStringSubmatch(href)
			if len(match) > 1 {
				if pageNum, err := strconv.Atoi(match[1]); err == nil {
					if pageNum > maxPage {
						maxPage = pageNum
					}
				}
			}
		}
	})

	if maxPage > 0 {
		return maxPage, nil
	}

	return 1, nil
}

func GetPoemLinks(pageNum int) ([]string, error) {
	url := fmt.Sprintf("%s&page=%d", baseURL, pageNum)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("페이지 %d 읽기 오류: %w", pageNum, err)
	}
	setHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("페이지 %d 읽기 오류: %w", pageNum, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("페이지 %d 읽기 오류: 상태 코드 %d", pageNum, resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("페이지 %d 읽기 오류: %w", pageNum, err)
	}

	var links []string
	seen := make(map[string]bool)

	doc.Find("a[href*=\"wr_id=\"]").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			if strings.Contains(href, "bo_table=tb01") && strings.Contains(href, "wr_id=") {
				parts := strings.Split(href, "wr_id=")
				if len(parts) > 1 {
					wrID := strings.Split(parts[1], "&")[0]
					if !seen[wrID] {
						seen[wrID] = true
						links = append(links, wrID)
					}
				}
			}
		}
	})

	return links, nil
}

type Poem struct {
	Title         string `json:"title" yaml:"title"`
	Author        string `json:"author" yaml:"author"`
	Content       string `json:"content" yaml:"content"`
	URL           string `json:"url" yaml:"url"`
	ReadingScript string `json:"reading_script" yaml:"reading_script"`
}

func getTextFromHTML(n *html.Node) string {
	var parts []string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			if len(node.Data) > 0 {
				parts = append(parts, node.Data)
			}
		} else if node.Type == html.ElementNode && node.Data == "br" {
			parts = append(parts, "\n")
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(parts, "\n")
}

func GetPoemDetail(wrID string) (*Poem, error) {
	url := fmt.Sprintf("%s&wr_id=%s", baseURL, wrID)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("상세 내용 가져오기 오류: %w", err)
	}
	setHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("상세 내용 가져오기 오류: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("상세 내용 가져오기 오류: 상태 코드 %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("상세 내용 가져오기 오류: %w", err)
	}

	poem := &Poem{
		URL: url,
	}

	titleTag := doc.Find("h1").First()
	if titleTag.Length() > 0 {
		poem.Title = stdhtml.UnescapeString(strings.TrimSpace(titleTag.Text()))
	} else {
		poem.Title = "제목 없음"
	}

	author := "미상"
	doc.Find("strong").EachWithBreak(func(i int, s *goquery.Selection) bool {
		parent := s.Parent()
		if parent.Length() > 0 && strings.Contains(parent.Text(), "저자") {
			author = stdhtml.UnescapeString(strings.TrimSpace(s.Text()))
			return false // break
		}
		return true // continue
	})
	poem.Author = author

	viewContent := doc.Find(".view-content").First()
	if viewContent.Length() > 0 {
		content := getTextFromHTML(viewContent.Get(0))

		lines := strings.Split(content, "\n")
		var cleanedLines []string
		for _, line := range lines {
			cleanedLines = append(cleanedLines, strings.TrimSpace(line))
		}
		content = strings.Join(cleanedLines, "\n")

		re := regexp.MustCompile(`\n{3,}`)
		content = strings.TrimSpace(re.ReplaceAllString(content, "\n\n"))

		if poem.Author == "미상" && strings.Contains(content, "저자 :") {
			parts := strings.Split(content, "저자 :")
			if len(parts) > 1 {
				words := strings.Fields(parts[1])
				if len(words) > 0 {
					poem.Author = strings.TrimSpace(words[0])
				}
			}
		}

		poem.Content = stdhtml.UnescapeString(content)
	} else {
		poem.Content = "내용을 불러올 수 없습니다."
	}

	return poem, nil
}
