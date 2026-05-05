package main

import (
	"strings"
	"testing"

	"github.com/suapapa/si-gnal/internal/poem"
)

func TestParseOutputFormat(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    outputFormat
		wantErr bool
	}{
		{name: "yaml default", in: "yaml", want: formatYAML},
		{name: "YAML trim", in: "  YAML  ", want: formatYAML},
		{name: "json", in: "json", want: formatJSON},
		{name: "txt", in: "txt", want: formatTXT},
		{name: "unknown", in: "xml", wantErr: true},
		{name: "empty", in: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOutputFormat(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatPoemTextFile(t *testing.T) {
	p := &poem.Poem{
		Title:  "제목",
		Author: "작가",
		Content: "본문",
		URL:    "https://example/poem",
	}
	s := formatPoemTextFile(p)
	if !containsAll(s, []string{"제목", "작가", "본문", "https://example/poem"}) {
		t.Fatalf("unexpected output: %q", s)
	}
}

func containsAll(s string, parts []string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
