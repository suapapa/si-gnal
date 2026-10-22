// Package openai implements the tts.TTS interface using the OpenAI
// (or OpenAI-compatible) audio/speech API.
package openai

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	openaisdk "github.com/sashabaranov/go-openai"
)

// Environment variables read by ConfigFromEnv.
const (
	EnvEndpoint = "OPENAI_ENDPOINT"
	EnvAPIKey   = "OPENAI_APIKEY"
	EnvTTSModel = "OPENAI_TTS_MODEL"
)

// Defaults applied when the corresponding Config field is empty.
const (
	DefaultEndpoint = "https://api.openai.com/v1"
	DefaultModel    = string(openaisdk.TTSModelGPT4oMini)
	DefaultVoice    = openaisdk.VoiceCoral
)

// OpenAI returns raw PCM from the speech API at 24kHz, 16-bit, mono.
const (
	pcmSampleRate   = 24000
	pcmNumChannels  = 1
	pcmBitDepth     = 16
	silenceDuration = 300 // ms of silence inserted between chunks
	// maxInputRunes keeps each speech request well under the API's
	// 4096-character input limit.
	maxInputRunes = 2000
)

// Config holds the settings for the OpenAI TTS engine.
type Config struct {
	// Endpoint is the OpenAI-compatible base URL (e.g. https://api.openai.com/v1).
	// Empty means DefaultEndpoint.
	Endpoint string
	// APIKey is required.
	APIKey string
	// Model is the TTS model (e.g. gpt-4o-mini-tts). Empty means DefaultModel.
	Model string
	// Voices lists the speaking voices to choose from. One is picked at
	// random for each EncodeWavIO call. Empty means DefaultVoice.
	Voices []string
	// Speed is the playback speed (0.25–4.0). Zero means 1.0.
	Speed float64
}

// ConfigFromEnv builds a Config from OPENAI_ENDPOINT, OPENAI_APIKEY and
// OPENAI_TTS_MODEL.
func ConfigFromEnv() Config {
	return Config{
		Endpoint: os.Getenv(EnvEndpoint),
		APIKey:   os.Getenv(EnvAPIKey),
		Model:    os.Getenv(EnvTTSModel),
	}
}

// TTS synthesizes speech through the OpenAI audio/speech API.
type TTS struct {
	client *openaisdk.Client
	model  string
	voices []openaisdk.SpeechVoice
	speed  float64
	// intn picks an index in [0, n); overridable in tests.
	intn func(n int) int
}

// NewTTS validates cfg and builds a TTS client.
func NewTTS(cfg Config) (*TTS, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("openai api key is empty (set %s)", EnvAPIKey)
	}
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = DefaultModel
	}
	voices := make([]openaisdk.SpeechVoice, 0, len(cfg.Voices))
	for _, v := range cfg.Voices {
		if v = strings.TrimSpace(v); v != "" {
			voices = append(voices, openaisdk.SpeechVoice(v))
		}
	}
	if len(voices) == 0 {
		voices = []openaisdk.SpeechVoice{DefaultVoice}
	}
	speed := cfg.Speed
	if speed == 0 {
		speed = 1.0
	}

	sdkCfg := openaisdk.DefaultConfig(cfg.APIKey)
	sdkCfg.BaseURL = endpoint

	return &TTS{
		client: openaisdk.NewClientWithConfig(sdkCfg),
		model:  model,
		voices: voices,
		speed:  speed,
		intn:   rand.IntN,
	}, nil
}

// pickVoice returns one of the configured voices at random, or
// DefaultVoice when none are configured.
func (t *TTS) pickVoice() openaisdk.SpeechVoice {
	if len(t.voices) == 0 {
		return DefaultVoice
	}
	if len(t.voices) == 1 {
		return t.voices[0]
	}
	intn := t.intn
	if intn == nil {
		intn = rand.IntN
	}
	return t.voices[intn(len(t.voices))]
}

// Close releases client resources (currently a no-op).
func (t *TTS) Close() {}

// EncodeWavIO synthesizes text and writes it to w as a WAV file
// (24kHz, 16-bit, mono PCM).
func (t *TTS) EncodeWavIO(w io.WriteSeeker, text string) error {
	return t.EncodeWavContext(context.Background(), w, text)
}

// EncodeWavContext is EncodeWavIO with a caller-supplied context.
func (t *TTS) EncodeWavContext(ctx context.Context, w io.WriteSeeker, text string) error {
	chunks := splitText(strings.TrimSpace(text))
	if len(chunks) == 0 {
		return fmt.Errorf("empty text")
	}

	// Pick one voice for the whole text so chunks of a long poem
	// are not read in different voices.
	voice := t.pickVoice()

	silence := make([]byte, pcmSampleRate*silenceDuration/1000*pcmBitDepth/8)
	var pcm []byte
	for i, chunk := range chunks {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context: %w", err)
		}
		data, err := t.synthesize(ctx, chunk, voice)
		if err != nil {
			return err
		}
		if len(data) == 0 {
			return fmt.Errorf("openai tts: empty audio for chunk %d", i)
		}
		if i > 0 {
			pcm = append(pcm, silence...)
		}
		pcm = append(pcm, data...)
	}

	if err := writeWAV(w, pcm); err != nil {
		return fmt.Errorf("write wav: %w", err)
	}
	return nil
}

func (t *TTS) synthesize(ctx context.Context, text string, voice openaisdk.SpeechVoice) ([]byte, error) {
	resp, err := t.client.CreateSpeech(ctx, openaisdk.CreateSpeechRequest{
		Model:          openaisdk.SpeechModel(t.model),
		Input:          text,
		Voice:          voice,
		ResponseFormat: openaisdk.SpeechResponseFormatPcm,
		Speed:          t.speed,
	})
	if err != nil {
		return nil, fmt.Errorf("CreateSpeech: %w", err)
	}
	defer func() { _ = resp.Close() }()

	data, err := io.ReadAll(resp)
	if err != nil {
		return nil, fmt.Errorf("read speech response: %w", err)
	}
	return data, nil
}

// splitText splits text into chunks of at most maxInputRunes runes,
// preferring sentence boundaries ('.', '!', '?', '…', newline).
func splitText(text string) []string {
	if text == "" {
		return nil
	}
	var sentences []string
	var cur strings.Builder
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			sentences = append(sentences, s)
		}
		cur.Reset()
	}
	for _, r := range text {
		cur.WriteRune(r)
		switch r {
		case '.', '!', '?', '…', '\n':
			flush()
		}
	}
	flush()

	var chunks []string
	var chunk strings.Builder
	curRunes := 0
	flushChunk := func() {
		if s := strings.TrimSpace(chunk.String()); s != "" {
			chunks = append(chunks, s)
		}
		chunk.Reset()
		curRunes = 0
	}
	for _, s := range sentences {
		n := utf8.RuneCountInString(s)
		if n > maxInputRunes {
			flushChunk()
			chunks = append(chunks, hardSplit(s, maxInputRunes)...)
			continue
		}
		// +1 for the space joining sentences.
		if curRunes > 0 && curRunes+1+n > maxInputRunes {
			flushChunk()
		}
		if curRunes > 0 {
			chunk.WriteByte(' ')
			curRunes++
		}
		chunk.WriteString(s)
		curRunes += n
	}
	flushChunk()
	return chunks
}

// hardSplit splits s into pieces of at most max runes, breaking at the
// last whitespace before the limit when possible.
func hardSplit(s string, max int) []string {
	var out []string
	runes := []rune(s)
	for len(runes) > max {
		cut := max
		for cut > 0 && !unicode.IsSpace(runes[cut]) {
			cut--
		}
		if cut == 0 {
			cut = max
		}
		if piece := strings.TrimSpace(string(runes[:cut])); piece != "" {
			out = append(out, piece)
		}
		runes = []rune(strings.TrimSpace(string(runes[cut:])))
	}
	if len(runes) > 0 {
		out = append(out, string(runes))
	}
	return out
}

// writeWAV writes pcm (24kHz, 16-bit, mono) to w with a WAV header.
func writeWAV(w io.Writer, pcm []byte) error {
	byteRate := uint32(pcmSampleRate * pcmNumChannels * pcmBitDepth / 8)
	blockAlign := uint16(pcmNumChannels * pcmBitDepth / 8)
	dataSize := uint32(len(pcm))

	if _, err := w.Write([]byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, 36+dataSize); err != nil {
		return err
	}
	if _, err := w.Write([]byte("WAVE")); err != nil {
		return err
	}
	if _, err := w.Write([]byte("fmt ")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(1)); err != nil { // PCM
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(pcmNumChannels)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(pcmSampleRate)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, byteRate); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, blockAlign); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(pcmBitDepth)); err != nil {
		return err
	}
	if _, err := w.Write([]byte("data")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, dataSize); err != nil {
		return err
	}
	_, err := w.Write(pcm)
	return err
}
