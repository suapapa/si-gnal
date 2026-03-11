package supertonic

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	ort "github.com/yalue/onnxruntime_go"
)

// Parameters holds command line arguments
type Parameters struct {
	UseGPU          bool
	ONNXDir         string
	TotalStep       int
	Speed           float32
	SilenceDuration float32
	VoiceStyles     []string
	Langs           []string
}

func NewDefaultParameters() *Parameters {
	cfgs := &Parameters{
		UseGPU:          false,
		ONNXDir:         "assets/supertonic2/onnx",
		TotalStep:       24, // 5~32 // 값이 높으면 발음이 훨씩 정확해지고 안정적임. 대신 오래걸림.
		Speed:           1.05,
		SilenceDuration: 0.3,
		VoiceStyles:     []string{"assets/supertonic2/voice_styles/F5.json"},
		Langs:           []string{"ko"},
	}

	return cfgs
}

type TTS struct {
	params       *Parameters
	onnxCfgs     *Config
	textToSpeech *TextToSpeech
	voiceStyle   *Style
}

func NewTTS(params *Parameters) (*TTS, error) {
	// Initialize ONNX Runtime
	if err := InitializeONNXRuntime(); err != nil {
		return nil, fmt.Errorf("error initializing ONNX Runtime: %v", err)
	}

	// Load config
	cfg, err := LoadCfgs(params.ONNXDir)
	if err != nil {
		return nil, fmt.Errorf("error loading config: %v", err)
	}

	// Load TTS components
	textToSpeech, err := LoadTextToSpeech(params.ONNXDir, params.UseGPU, cfg)
	if err != nil {
		return nil, fmt.Errorf("error loading TTS components: %v", err)
	}

	style, err := LoadVoiceStyle(params.VoiceStyles, true)
	if err != nil {
		return nil, fmt.Errorf("error loading voice styles: %v", err)
	}

	return &TTS{
		params:       params,
		onnxCfgs:     &cfg,
		textToSpeech: textToSpeech,
		voiceStyle:   style,
	}, nil
}

func (e *TTS) Close() {
	if e.voiceStyle != nil {
		e.voiceStyle.Destroy()
	}
	if e.textToSpeech != nil {
		e.textToSpeech.Destroy()
	}
	ort.DestroyEnvironment()
}

func (e *TTS) EncodeWavIO(w io.WriteSeeker, text string) error {
	wav, duration, err := e.textToSpeech.Call(text, e.params.Langs[0], e.voiceStyle, e.params.TotalStep, e.params.Speed, e.params.SilenceDuration)
	if err != nil {
		return fmt.Errorf("error generating speech: %w", err)
	}

	var wavOut []float64

	// For non-batch mode, wav is a single concatenated audio
	wavLen := int(float32(e.textToSpeech.SampleRate) * duration)
	wavOut = make([]float64, wavLen)
	for j := 0; j < wavLen && j < len(wav); j++ {
		wavOut[j] = float64(wav[j])
	}

	if err := writeWavFileIO(w, wavOut, e.textToSpeech.SampleRate); err != nil {
		return fmt.Errorf("error writing wav file: %w", err)
	}

	return nil
}

func (e *TTS) BatchEncodeToFiles(saveDir string, texts []string) error {
	// --- 5. Synthesize speech --- //
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("error creating save directory: %w", err)
	}

	var wav []float32
	var duration []float32

	w, d, err := e.textToSpeech.Batch(texts, e.params.Langs, e.voiceStyle, e.params.TotalStep, e.params.Speed)
	if err != nil {
		return fmt.Errorf("error generating batch speech: %w", err)
	}
	wav = w
	duration = d

	// Save outputs
	for i := 0; i < len(texts); i++ {
		fname := fmt.Sprintf("%s.wav", sanitizeFilename(texts[i], 20))
		var wavOut []float64

		wavOut = extractWavSegment(wav, duration[i], e.textToSpeech.SampleRate, i, len(texts))

		outputPath := filepath.Join(saveDir, fname)
		if err := writeWavFile(outputPath, wavOut, e.textToSpeech.SampleRate); err != nil {
			return fmt.Errorf("error writing wav file: %w", err)
		}
	}

	return nil
}
