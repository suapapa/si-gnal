package ai

import "testing"

func TestNormalizeReadingScript(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "hanja annotation after hangul", in: "산보로(散步路)를 걷는다", want: "산보로를 걷는다"},
		{name: "fullwidth hanja annotation", in: "고향（故鄕）으로 간다", want: "고향으로 간다"},
		{name: "hanja with hangul reading", in: "散步路(산보로)를 걷는다", want: "산보로를 걷는다"},
		{name: "duplicated hangul in parens", in: "산보로(산보로)를 걷는다", want: "산보로를 걷는다"},
		{name: "space before annotation", in: "산보로 (散步路) 를 걷는다", want: "산보로 를 걷는다"},
		{name: "multiple annotations", in: "마음(心)이 고향(故鄕)을 향한다", want: "마음이 고향을 향한다"},
		{name: "real parenthetical content kept", in: "그는 (아무 말 없이) 떠났다", want: "그는 (아무 말 없이) 떠났다"},
		{name: "no parens", in: "산보로를 걷는다", want: "산보로를 걷는다"},
		{name: "unclosed paren kept", in: "산보로(散步路", want: "산보로(散步路"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeReadingScript(tt.in); got != tt.want {
				t.Errorf("normalizeReadingScript(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
