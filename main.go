package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/suapapa/signal/internal/poem"
	"github.com/suapapa/signal/internal/poem/ai"
	wirephone_sound "github.com/suapapa/signal/internal/sound/wirephone"
	"github.com/suapapa/signal/internal/tts"
	"github.com/suapapa/signal/internal/tts/htgo"
	"github.com/suapapa/signal/internal/tts/supertonic"
)

var (
	batch      int
	enticSound bool
	addNoise   bool
	ttsEngine  string
)

func main() {
	flag.IntVar(&batch, "b", 1, "batch count")
	flag.BoolVar(&enticSound, "e", false, "apply wire-phone effect on output")
	flag.BoolVar(&addNoise, "n", false, "add noise")
	flag.StringVar(&ttsEngine, "t", "supertonic", "tts engine (supertonic, htgo, melotts)")
	flag.Parse()

	// init engines

	var t tts.TTS
	var err error

	switch ttsEngine {
	case "supertonic":
		ttsParams := supertonic.NewDefaultParameters()
		ttsParams.TotalStep = 32
		ttsParams.Speed = 0.85
		ttsParams.SilenceDuration = 1.2
		t, err = supertonic.NewTTS(ttsParams)
	case "htgo":
		t, err = htgo.NewTTS("ko")
	default:
		log.Fatalf("unknown tts engine: %s", ttsEngine)
	}
	if err != nil {
		log.Fatalf("failed to init TTS: %v", err)
	}
	defer t.Close()

	aiFix, err := ai.NewAI(context.Background())
	if err != nil {
		log.Fatalf("failed to init AI: %v", err)
	}
	defer aiFix.Close()

	for i := 0; i < batch; i++ {
		// Fetch a random poem
		p := fetchRandomPoem()

		// print title and author of the poem
		yamlPoem, err := yaml.Marshal(p)
		if err != nil {
			log.Fatal("faile to generate yaml for poem")
		}
		os.Stdout.Write(yamlPoem)

		// cleanup poem content
		if err := aiFix.CleanupContent(context.Background(), p); err != nil {
			log.Printf("CleanupContent failed: %v", err)
		}

		// fix poem content for tts
		if err := aiFix.FixContentForTTS(context.Background(), p); err != nil {
			log.Printf("FixContentForTTS failed: %v", err)
		}

		// print poem content
		fmt.Println("Poem Content:")
		fmt.Println(p.Content)

		// make tts wav for poem content
		fmt.Println("Generating TTS...")

		tempWav := fmt.Sprintf("poem_%03d.wav", i+1)
		wavFile, err := os.Create(tempWav)
		if err != nil {
			log.Fatalf("failed to create %s: %v", tempWav, err)
		}

		if err := t.EncodeWavIO(wavFile, p.Content); err != nil {
			wavFile.Close()
			log.Fatalf("failed to encode wav: %v", err)
		}

		// if err := t.BatchEncodeToFiles("poems", strings.Split(p.Content, "\n")); err != nil {
		// 	log.Fatalf("failed to encode batch wav: %v", err)
		// }

		if enticSound {
			// apply wirephone effect
			fmt.Println("Applying wirephone effect...")
			if _, err := wavFile.Seek(0, 0); err != nil {
				wavFile.Close()
				log.Fatalf("failed to seek wav file: %v", err)
			}

			const effectWav = "poem_wirephone.wav"
			outWavFile, err := os.Create(effectWav)
			if err != nil {
				wavFile.Close()
				log.Fatalf("failed to create output wav file: %v", err)
			}

			err = wirephone_sound.MakeAntiquePhone(wavFile, outWavFile, addNoise)
			wavFile.Close()
			outWavFile.Close()

			if err != nil {
				log.Fatalf("failed to apply wirephone effect: %v", err)
			}

			// replace original with the one with effect
			if err := os.Rename(effectWav, tempWav); err != nil {
				log.Fatalf("failed to replace file: %v", err)
			}
		} else {
			wavFile.Close()
		}

		fmt.Printf("Successfully created %s\n\n", tempWav)
	}
}

func fetchRandomPoem() *poem.Poem {
	lastPage, err := poem.GetLastPage()
	if err != nil {
		log.Fatalf("failed to get last page: %v", err)
	}

	page := rand.Intn(lastPage) + 1
	links, err := poem.GetPoemLinks(page)
	if err != nil {
		log.Fatalf("failed to get poem links: %v", err)
	}
	if len(links) == 0 {
		log.Fatalf("no poems found on page %d", page)
	}

	link := links[rand.Intn(len(links))]
	p, err := poem.GetPoemDetail(link)
	if err != nil {
		log.Fatalf("failed to get poem detail: %v", err)
	}

	return p
}
