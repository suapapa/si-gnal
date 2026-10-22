package main

import (
	"flag"
	"log"
	"os"

	openaits "github.com/suapapa/si-gnal/internal/tts/openai"
)

var (
	inputText = "지나온 모든 계절은 너를 위한 배경이었고, 이제야 비로소 네가 선명해진 계절이 왔다."

	inputFile string
)

func main() {
	flag.StringVar(&inputFile, "i", "", "input file")
	flag.Parse()

	if inputFile != "" {
		b, err := os.ReadFile(inputFile)
		if err != nil {
			log.Fatalf("Failed to read input file: %v", err)
		}
		inputText = string(b)
	}

	// OpenAI settings come from OPENAI_ENDPOINT / OPENAI_APIKEY /
	// OPENAI_TTS_MODEL (see .env.example).
	ttsEngine, err := openaits.NewTTS(openaits.ConfigFromEnv())
	if err != nil {
		log.Fatalf("Failed to initialize OpenAI TTS engine: %v", err)
	}
	defer ttsEngine.Close()

	outputFile := "output.wav"
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer f.Close()

	log.Printf("Generating speech for: %q using OpenAI TTS...\n", inputText)
	if err := ttsEngine.EncodeWavIO(f, inputText); err != nil {
		log.Fatalf("Failed to encode wav: %v", err)
	}

	log.Printf("Successfully generated speech and saved to %s\n", outputFile)
}
