package openai

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// memWriteSeeker is an in-memory io.WriteSeeker for tests.
type memWriteSeeker struct {
	buf []byte
	pos int64
}

func (m *memWriteSeeker) Write(p []byte) (int, error) {
	end := m.pos + int64(len(p))
	if end > int64(len(m.buf)) {
		grown := make([]byte, end)
		copy(grown, m.buf)
		m.buf = grown
	}
	copy(m.buf[m.pos:], p)
	m.pos = end
	return len(p), nil
}

func (m *memWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	var pos int64
	switch whence {
	case io.SeekStart:
		pos = offset
	case io.SeekCurrent:
		pos = m.pos + offset
	case io.SeekEnd:
		pos = int64(len(m.buf)) + offset
	}
	m.pos = pos
	return pos, nil
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv(EnvEndpoint, "https://example.com/v1")
	t.Setenv(EnvAPIKey, "sk-test")
	t.Setenv(EnvTTSModel, "tts-1")

	cfg := ConfigFromEnv()
	if cfg.Endpoint != "https://example.com/v1" || cfg.APIKey != "sk-test" || cfg.Model != "tts-1" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestNewTTSRequiresAPIKey(t *testing.T) {
	if _, err := NewTTS(Config{}); err == nil {
		t.Fatal("expected error for empty API key")
	}
}

func TestEncodeWavIO(t *testing.T) {
	pcm := []byte{0x01, 0x00, 0xff, 0xff, 0x00, 0x00}
	var gotModel, gotVoice, gotFormat, gotInstructions string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/speech" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer sk-test" {
			t.Errorf("unexpected authorization: %q", auth)
		}
		var req struct {
			Model        string `json:"model"`
			Input        string `json:"input"`
			Voice        string `json:"voice"`
			Format       string `json:"response_format"`
			Instructions string `json:"instructions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		gotModel, gotVoice, gotFormat, gotInstructions = req.Model, req.Voice, req.Format, req.Instructions
		if req.Input == "" {
			t.Errorf("empty input")
		}
		w.Header().Set("Content-Type", "audio/pcm")
		_, _ = w.Write(pcm)
	}))
	defer srv.Close()

	tts, err := NewTTS(Config{Endpoint: srv.URL, APIKey: "sk-test", Model: "gpt-4o-mini-tts"})
	if err != nil {
		t.Fatalf("NewTTS: %v", err)
	}
	defer tts.Close()

	out := &memWriteSeeker{}
	if err := tts.EncodeWavIO(out, "안녕하세요. 시를 읽어 드립니다."); err != nil {
		t.Fatalf("EncodeWavIO: %v", err)
	}

	if gotModel != "gpt-4o-mini-tts" {
		t.Errorf("model = %q, want gpt-4o-mini-tts", gotModel)
	}
	if gotVoice != string(DefaultVoice) {
		t.Errorf("voice = %q, want %q", gotVoice, DefaultVoice)
	}
	if gotFormat != "pcm" {
		t.Errorf("response_format = %q, want pcm", gotFormat)
	}
	if gotInstructions != "In a calm poetry recitation tone." {
		t.Errorf("instructions = %q, want %q", gotInstructions, "In a calm poetry recitation tone.")
	}

	// Sentences are joined into one chunk, so the WAV payload is just pcm.
	if !bytes.HasPrefix(out.buf, []byte("RIFF")) || !bytes.Contains(out.buf, []byte("WAVE")) {
		t.Fatalf("missing WAV header: %q", out.buf[:min(16, len(out.buf))])
	}
	dataSize := binary.LittleEndian.Uint32(out.buf[40:44])
	if int(dataSize) != len(pcm) {
		t.Errorf("data size = %d, want %d", dataSize, len(pcm))
	}
	if !bytes.Equal(out.buf[44:], pcm) {
		t.Errorf("pcm payload mismatch: got %v, want %v", out.buf[44:], pcm)
	}
	if sr := binary.LittleEndian.Uint32(out.buf[24:28]); sr != pcmSampleRate {
		t.Errorf("sample rate = %d, want %d", sr, pcmSampleRate)
	}
}

func TestEncodeWavIO_PicksOneVoiceFromList(t *testing.T) {
	pcm := []byte{0x01, 0x00}
	var gotVoices []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Voice string `json:"voice"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		gotVoices = append(gotVoices, req.Voice)
		_, _ = w.Write(pcm)
	}))
	defer srv.Close()

	tts, err := NewTTS(Config{
		Endpoint: srv.URL,
		APIKey:   "sk-test",
		Voices:   []string{"alloy", " nova ", "", "shimmer"},
	})
	if err != nil {
		t.Fatalf("NewTTS: %v", err)
	}
	defer tts.Close()
	// Deterministically pick the last voice (whitespace/empties are dropped,
	// so the list is [alloy nova shimmer]).
	tts.intn = func(n int) int { return n - 1 }

	// Long enough to be split into multiple /audio/speech requests.
	text := strings.Repeat("가", maxInputRunes+100) + "."
	out := &memWriteSeeker{}
	if err := tts.EncodeWavIO(out, text); err != nil {
		t.Fatalf("EncodeWavIO: %v", err)
	}
	if len(gotVoices) < 2 {
		t.Fatalf("expected multiple speech requests, got %d", len(gotVoices))
	}
	for i, v := range gotVoices {
		if v != "shimmer" {
			t.Errorf("request %d voice = %q, want shimmer (one voice per text)", i, v)
		}
	}
}

func TestPickVoice(t *testing.T) {
	tts, err := NewTTS(Config{APIKey: "sk-test"})
	if err != nil {
		t.Fatalf("NewTTS: %v", err)
	}
	if got := tts.pickVoice(); got != DefaultVoice {
		t.Errorf("pickVoice with no voices = %q, want %q", got, DefaultVoice)
	}

	single, err := NewTTS(Config{APIKey: "sk-test", Voices: []string{"nova"}})
	if err != nil {
		t.Fatalf("NewTTS: %v", err)
	}
	if got := single.pickVoice(); got != "nova" {
		t.Errorf("pickVoice with one voice = %q, want nova", got)
	}
}

func TestSplitText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty", in: "  ", want: nil},
		{name: "single", in: "안녕하세요.", want: []string{"안녕하세요."}},
		{
			name: "sentences joined",
			in:   "첫 문장입니다. 둘째 문장입니다! 셋째인가요?",
			want: []string{"첫 문장입니다. 둘째 문장입니다! 셋째인가요?"},
		},
		{
			name: "newline boundary",
			in:   "첫 줄\n둘째 줄",
			want: []string{"첫 줄 둘째 줄"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := splitText(tc.in)
			if strings.Join(got, "\x00") != strings.Join(tc.want, "\x00") {
				t.Errorf("splitText(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSplitTextLong(t *testing.T) {
	long := strings.Repeat("가", maxInputRunes+100) + "."
	chunks := splitText(long)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	for i, c := range chunks {
		if n := len([]rune(c)); n > maxInputRunes {
			t.Errorf("chunk %d has %d runes, want <= %d", i, n, maxInputRunes)
		}
	}
}
