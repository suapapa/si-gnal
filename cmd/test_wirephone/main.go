package main

import (
	"fmt"
	"log"
	"os"

	wirephone_sound "github.com/suapapa/signal/internal/sound/wirephone"
)

func main() {
	inPath := "output.wav"
	outPath := "output_wirephone.wav"

	inFile, err := os.Open(inPath)
	if err != nil {
		log.Fatal(err)
	}
	defer inFile.Close()

	outFile, err := os.Create(outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	fmt.Println("Applying wirephone effect...")
	if err := wirephone_sound.MakeAntiquePhone(inFile, outFile, false); err != nil {
		log.Fatal("Effect failed:", err)
	}
	fmt.Println("Done! Saved to", outPath)
}
