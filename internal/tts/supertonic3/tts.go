package supertonic

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ort "github.com/yalue/onnxruntime_go"
)

// Parameters holds TTS engine configuration.
type Parameters struct {
	UseGPU          bool
	ONNXDir         string
	TotalStep       int
	Speed           float32
	SilenceDuration float32
	VoiceStyles     []string
	Langs           []string
}

// NewDefaultParameters returns sensible defaults for Korean poem TTS.
func NewDefaultParameters() *Parameters {
	return &Parameters{
		UseGPU:          false,
		ONNXDir:         "assets/supertonic3/onnx",
		TotalStep:       24, // 5~32; higher is more accurate/stable but slower
		Speed:           1.05,
		SilenceDuration: 0.3,
		VoiceStyles:     []string{"assets/supertonic3/voice_styles/F5.json"},
		Langs:           []string{"ko"},
	}
}

// TTS wraps the official Supertonic ONNX pipeline behind the project TTS interface.
type TTS struct {
	params       *Parameters
	onnxCfgs     *Config
	textToSpeech *TextToSpeech
	voiceStyle   *Style
}

// NewTTS initializes ONNX Runtime, loads models, and prepares a voice style.
func NewTTS(params *Parameters) (*TTS, error) {
	if err := InitializeONNXRuntime(); err != nil {
		return nil, fmt.Errorf("error initializing ONNX Runtime: %w", err)
	}

	cfg, err := LoadCfgs(params.ONNXDir)
	if err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}

	textToSpeech, err := LoadTextToSpeech(params.ONNXDir, params.UseGPU, cfg)
	if err != nil {
		return nil, fmt.Errorf("error loading TTS components: %w", err)
	}

	style, err := LoadVoiceStyle(params.VoiceStyles, true)
	if err != nil {
		return nil, fmt.Errorf("error loading voice styles: %w", err)
	}

	return &TTS{
		params:       params,
		onnxCfgs:     &cfg,
		textToSpeech: textToSpeech,
		voiceStyle:   style,
	}, nil
}

// Close releases ONNX sessions and the runtime environment.
func (e *TTS) Close() {
	if e.voiceStyle != nil {
		e.voiceStyle.Destroy()
	}
	if e.textToSpeech != nil {
		e.textToSpeech.Destroy()
	}
	ort.DestroyEnvironment()
}

// poemSegments splits on '.' (ASCII full stop). Empty pieces are dropped.
// If nothing remains (e.g. only dots), the original trimmed text is used as one segment.
func poemSegments(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	parts := strings.Split(text, ".")
	var segs []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		segs = append(segs, p+".")
	}
	if len(segs) == 0 {
		return []string{text}
	}
	return segs
}

// EncodeWavIO synthesizes speech for text and writes a WAV to w.
func (e *TTS) EncodeWavIO(w io.WriteSeeker, text string) error {
	segments := poemSegments(text)
	if len(segments) == 0 {
		return fmt.Errorf("empty text")
	}

	sr := e.textToSpeech.SampleRate
	silenceSamples := int(float32(sr) * e.params.SilenceDuration)
	lang := e.params.Langs[0]

	var combined []float64
	for i, seg := range segments {
		wav, duration, err := e.textToSpeech.Call(seg, lang, e.voiceStyle, e.params.TotalStep, e.params.Speed, e.params.SilenceDuration)
		if err != nil {
			return fmt.Errorf("error generating speech: %w", err)
		}

		wavLen := int(float32(sr) * duration)
		if wavLen > len(wav) {
			wavLen = len(wav)
		}

		if i > 0 && silenceSamples > 0 {
			combined = append(combined, make([]float64, silenceSamples)...)
		}
		chunk := make([]float64, wavLen)
		for j := 0; j < wavLen; j++ {
			chunk[j] = float64(wav[j])
		}
		combined = append(combined, chunk...)
	}

	if err := writeWavFileIO(w, combined, sr); err != nil {
		return fmt.Errorf("error writing wav file: %w", err)
	}

	return nil
}

// BatchEncodeToFiles synthesizes multiple texts and writes one WAV per text.
func (e *TTS) BatchEncodeToFiles(saveDir string, texts []string) error {
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("error creating save directory: %w", err)
	}

	wav, duration, err := e.textToSpeech.Batch(texts, e.params.Langs, e.voiceStyle, e.params.TotalStep, e.params.Speed)
	if err != nil {
		return fmt.Errorf("error generating batch speech: %w", err)
	}

	for i := 0; i < len(texts); i++ {
		fname := fmt.Sprintf("%s.wav", sanitizeFilename(texts[i], 20))
		wavOut := extractWavSegment(wav, duration[i], e.textToSpeech.SampleRate, i, len(texts))

		outputPath := filepath.Join(saveDir, fname)
		if err := writeWavFile(outputPath, wavOut, e.textToSpeech.SampleRate); err != nil {
			return fmt.Errorf("error writing wav file: %w", err)
		}
	}

	return nil
}
