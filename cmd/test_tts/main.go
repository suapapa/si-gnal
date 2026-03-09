package main

import (
	"log"
	"os"

	"github.com/suapapa/signal/internal/tts"
)

var (
	inputText = "지나온 모든 계절은 너를 위한 배경이었고, 이제야 비로소 네가 선명해진 계절이 왔다."
)

func main() {
	// 1. Get default parameters for TTS
	params := tts.NewDefaultParameters()
	params.ONNXDir = "../../assets/onnx"
	params.VoiceStyles = []string{
		"../../assets/voice_styles/F5.json",
	}

	if len(os.Args) > 1 {
		inputText = os.Args[1]
	}

	// 2. Initialize the TTS engine
	// Make sure that ONNX runtime and models (assets/onnx, assets/voice_styles/F5.json) are available in the path.
	ttsEngine, err := tts.NewTTS(params)
	if err != nil {
		log.Fatalf("Failed to initialize TTS engine: %v", err)
	}
	defer ttsEngine.Close()

	// 3. Create an output file that implements io.WriteSeeker
	outputFile := "output.wav"
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer f.Close()

	// 4. Encode the text and write it as WAV to the file
	log.Printf("Generating speech for: %q...\n", inputText)
	err = ttsEngine.EncodeWavIO(f, inputText)
	if err != nil {
		log.Fatalf("Failed to encode wav: %v", err)
	}

	log.Printf("Successfully generated speech and saved to %s\n", outputFile)
}
