package poem

import (
	"context"
	"testing"
)

func TestGetPoemDetail(t *testing.T) {
	p, err := GetPoemDetail(context.Background(), "27235")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Title:\n" + p.Title)
	t.Log("Author:\n" + p.Author)
	t.Log("Content:\n" + p.Content)
}
