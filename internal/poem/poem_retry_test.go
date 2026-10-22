package poem

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// dropConnection simulates poemlove.co.kr closing the connection without
// a response, which the client reports as EOF.
func dropConnection(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	hj, ok := w.(http.Hijacker)
	if !ok {
		t.Fatal("ResponseWriter does not support hijacking")
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		t.Fatalf("hijack: %v", err)
	}
	_ = conn.Close()
}

// useTestServer points baseURL at srv and shrinks the retry backoff.
func useTestServer(t *testing.T, srv *httptest.Server) {
	t.Helper()
	oldURL, oldAttempts, oldBackoff := baseURL, maxFetchAttempts, fetchRetryBackoff
	baseURL = srv.URL + "?bo_table=tb01"
	maxFetchAttempts = 4
	fetchRetryBackoff = time.Millisecond
	t.Cleanup(func() {
		baseURL, maxFetchAttempts, fetchRetryBackoff = oldURL, oldAttempts, oldBackoff
	})
}

const linksPage = `<html><body>
<a href="board.php?bo_table=tb01&wr_id=111">시 1</a>
<a href="board.php?bo_table=tb01&wr_id=222">시 2</a>
</body></html>`

func TestGetPoemLinks_RetriesOnEOF(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 { // drop the first two connections
			dropConnection(t, w)
			return
		}
		_, _ = w.Write([]byte(linksPage))
	}))
	defer srv.Close()
	useTestServer(t, srv)

	links, err := GetPoemLinks(context.Background(), 1049)
	if err != nil {
		t.Fatalf("GetPoemLinks: %v", err)
	}
	if len(links) != 2 || links[0] != "111" || links[1] != "222" {
		t.Fatalf("links = %v, want [111 222]", links)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("requests = %d, want 3", got)
	}
}

func TestGetPoemLinks_RetriesOn5xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(linksPage))
	}))
	defer srv.Close()
	useTestServer(t, srv)

	links, err := GetPoemLinks(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetPoemLinks: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("links = %v, want 2 links", links)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("requests = %d, want 3", got)
	}
}

func TestGetPoemLinks_NoRetryOn404(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	useTestServer(t, srv)

	_, err := GetPoemLinks(context.Background(), 99999)
	if err == nil {
		t.Fatal("GetPoemLinks: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "상태 코드 404") {
		t.Fatalf("error = %q, want it to mention 상태 코드 404", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1 (no retry on 404)", got)
	}
}

func TestGetPoemLinks_GivesUpAfterMaxAttempts(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		dropConnection(t, w)
	}))
	defer srv.Close()
	useTestServer(t, srv)

	_, err := GetPoemLinks(context.Background(), 1049)
	if err == nil {
		t.Fatal("GetPoemLinks: expected error, got nil")
	}
	if got := calls.Load(); got != int32(maxFetchAttempts) {
		t.Fatalf("requests = %d, want %d", got, maxFetchAttempts)
	}
}

func TestFetch_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dropConnection(t, w)
	}))
	defer srv.Close()
	useTestServer(t, srv)
	fetchRetryBackoff = time.Hour // would block forever without ctx handling

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := GetPoemLinks(ctx, 1)
	if err == nil {
		t.Fatal("GetPoemLinks: expected error, got nil")
	}
}
