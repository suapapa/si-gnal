package main

import (
	"flag"
	"log"
	"os"

	"github.com/suapapa/si-gnal/internal/tts"
	"github.com/suapapa/si-gnal/internal/tts/htgo"
	"github.com/suapapa/si-gnal/internal/tts/supertonic"
)

var (
	inputText = "지나온 모든 계절은 너를 위한 배경이었고, 이제야 비로소 네가 선명해진 계절이 왔다."
)

func main() {
	engineFlag := flag.String("engine", "supertonic", "TTS engine to use: htgo, supertonic")
	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		inputText = args[0]
	}

	var ttsEngine tts.TTS
	var err error

	// 1 & 2. Initialize the TTS engine based on the selected flag
	switch *engineFlag {
	case "htgo":
		ttsEngine, err = htgo.NewTTS("ko")
	case "supertonic":
		params := supertonic.NewDefaultParameters()
		params.TotalStep = 32
		params.ONNXDir = "../../assets/onnx"
		params.Speed = 0.85
		params.SilenceDuration = 1.2
		params.VoiceStyles = []string{
			"../../assets/voice_styles/F5.json",
		}
		ttsEngine, err = supertonic.NewTTS(params)
	default:
		log.Fatalf("Unknown engine: %s. Available options: htgo, melotts, supertonic", *engineFlag)
	}

	if err != nil {
		log.Fatalf("Failed to initialize TTS engine %s: %v", *engineFlag, err)
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
	log.Printf("Generating speech for: %q using engine %s...\n", inputText, *engineFlag)
	err = ttsEngine.EncodeWavIO(f, inputText)
	if err != nil {
		log.Fatalf("Failed to encode wav: %v", err)
	}

	log.Printf("Successfully generated speech and saved to %s\n", outputFile)
}
