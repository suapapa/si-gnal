package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/suapapa/si-gnal/internal/sound/dtmf"
)

func main() {
	input := flag.String("input", "1004#", "DTMF string to generate (e.g., 1004#)")
	output := flag.String("out", "output.wav", "Output WAV file name")
	flag.Parse()

	if *input == "" {
		log.Fatal("input string cannot be empty")
	}

	fmt.Printf("Generating DTMF recording for %q...\n", *input)

	// Create the output file
	f, err := os.Create(*output)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	// Generate WAV and write to the file
	err = dtmf.GenerateWav(*input, f)
	if err != nil {
		log.Fatalf("Failed to generate DTMF WAV: %v", err)
	}

	fmt.Printf("Successfully generated %s\n", *output)
}
