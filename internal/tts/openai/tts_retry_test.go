package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// useRetryTestServer shrinks the speech retry backoff for srv-based tests.
func useRetryTestServer(t *testing.T) {
	t.Helper()
	oldAttempts, oldBackoff := maxSpeechAttempts, speechRetryBackoff
	maxSpeechAttempts = 4
	speechRetryBackoff = time.Millisecond
	t.Cleanup(func() {
		maxSpeechAttempts, speechRetryBackoff = oldAttempts, oldBackoff
	})
}

func newRetryTestTTS(t *testing.T, srv *httptest.Server) *TTS {
	t.Helper()
	tts, err := NewTTS(Config{Endpoint: srv.URL, APIKey: "sk-test"})
	if err != nil {
		t.Fatalf("NewTTS: %v", err)
	}
	t.Cleanup(tts.Close)
	return tts
}

// writeProviderError mimics an OpenAI-compatible provider answering 500
// with a JSON error body: "provider API error (status 500)".
func writeProviderError(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"error":{"message":"provider API error (status 500)","type":"server_error"}}`))
}

func TestEncodeWavIO_RetriesOn500(t *testing.T) {
	pcm := []byte{0x01, 0x00, 0xff, 0xff}
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 { // fail the first two attempts with 500
			writeProviderError(w, http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(pcm)
	}))
	defer srv.Close()
	useRetryTestServer(t)

	out := &memWriteSeeker{}
	if err := newRetryTestTTS(t, srv).EncodeWavIO(out, "안녕하세요."); err != nil {
		t.Fatalf("EncodeWavIO: %v", err)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("requests = %d, want 3", got)
	}
	if len(out.buf) == 0 {
		t.Fatal("empty wav output")
	}
}

func TestEncodeWavIO_RetriesOnPlain502(t *testing.T) {
	// Non-JSON error bodies surface as RequestError, not APIError.
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("bad gateway"))
			return
		}
		_, _ = w.Write([]byte{0x01, 0x00})
	}))
	defer srv.Close()
	useRetryTestServer(t)

	out := &memWriteSeeker{}
	if err := newRetryTestTTS(t, srv).EncodeWavIO(out, "안녕하세요."); err != nil {
		t.Fatalf("EncodeWavIO: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("requests = %d, want 2", got)
	}
}

func TestEncodeWavIO_NoRetryOn400(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeProviderError(w, http.StatusBadRequest)
	}))
	defer srv.Close()
	useRetryTestServer(t)

	out := &memWriteSeeker{}
	err := newRetryTestTTS(t, srv).EncodeWavIO(out, "안녕하세요.")
	if err == nil {
		t.Fatal("EncodeWavIO: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "status code: 400") {
		t.Fatalf("error = %q, want it to mention status code 400", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1 (no retry on 400)", got)
	}
}

func TestEncodeWavIO_GivesUpAfterMaxAttempts(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeProviderError(w, http.StatusInternalServerError)
	}))
	defer srv.Close()
	useRetryTestServer(t)

	out := &memWriteSeeker{}
	if err := newRetryTestTTS(t, srv).EncodeWavIO(out, "안녕하세요."); err == nil {
		t.Fatal("EncodeWavIO: expected error, got nil")
	}
	if got := calls.Load(); got != int32(maxSpeechAttempts) {
		t.Fatalf("requests = %d, want %d", got, maxSpeechAttempts)
	}
}

func TestEncodeWavIO_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cancel() // cancel while synthesize is backing off before retrying
		writeProviderError(w, http.StatusInternalServerError)
	}))
	defer srv.Close()
	useRetryTestServer(t)
	speechRetryBackoff = time.Hour // would block forever without ctx handling

	out := &memWriteSeeker{}
	if err := newRetryTestTTS(t, srv).EncodeWavContext(ctx, out, "안녕하세요."); err == nil {
		t.Fatal("EncodeWavContext: expected error, got nil")
	}
}
