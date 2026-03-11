package main

import (
	"fmt"
	"log"
	"os"

	"github.com/suapapa/si-gnal/internal/player"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test_player <wav_file_path>")
		os.Exit(1)
	}

	wavFilePath := os.Args[1]

	log.Printf("Playing %s...", wavFilePath)
	if err := player.PlayWav(wavFilePath); err != nil {
		log.Fatalf("failed to play %s: %v", wavFilePath, err)
	}
	log.Println("Done")
}
