package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/suapapa/si-gnal/internal/player"
	"github.com/suapapa/si-gnal/internal/poem"
	"github.com/suapapa/si-gnal/internal/poem/ai"
	wirephone_sound "github.com/suapapa/si-gnal/internal/sound/wirephone"
	"github.com/suapapa/si-gnal/internal/tts"
	"github.com/suapapa/si-gnal/internal/tts/htgo"
	"github.com/suapapa/si-gnal/internal/tts/supertonic"
)

var (
	batch      int
	enticSound bool
	addNoise   bool
	ttsEngine  string
	playOutput bool
)

func main() {
	flag.IntVar(&batch, "b", 5, "batch count for pre-generated wav files")
	flag.BoolVar(&enticSound, "e", false, "apply wire-phone effect on output")
	flag.BoolVar(&addNoise, "n", false, "add noise")
	flag.StringVar(&ttsEngine, "t", "supertonic", "tts engine (supertonic, htgo)")
	flag.BoolVar(&playOutput, "p", false, "play the output wav file directly to speaker (legacy)")
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

	type PlayJob struct {
		WavName string     `json:"wavName"`
		Poem    *poem.Poem `json:"poem"`
	}
	wavQueue := make(chan PlayJob, batch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() {
		log.Println("🧹 대기 큐 정리중...")
		close(wavQueue)
		for job := range wavQueue {
			os.Remove(job.WavName)
		}
		log.Println("👋 ByeBye")
	}()

	log.Printf("🚀 Starting background generator (batch size: %d)...\n", batch)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			p, wavFile, err := generateOneWav(ctx, t, aiFix)
			if err != nil {
				log.Printf("failed to generate wav: %v", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Minute):
				}
				continue
			}

			select {
			case <-ctx.Done():
				os.Remove(wavFile)
				return
			case wavQueue <- PlayJob{WavName: wavFile, Poem: p}: // Blocks if queue is full
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		select {
		case job := <-wavQueue:
			go func() {
				log.Printf("▶️ Playing %s...", job.WavName)
				if err := player.PlayWav(job.WavName); err != nil {
					log.Printf("failed to play %s: %v", job.WavName, err)
				}
				os.Remove(job.WavName) // Cleanup after playing
			}()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(job)
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "Queue is empty, try again later")
		}
	})

	port := ":8080"
	srv := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	log.Printf("🌐 Starting web server on %s...", port)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Graceful shutdown setup
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")

	// Cancel background generator
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exiting")
}

func generateOneWav(ctx context.Context, t tts.TTS, aiFix *ai.AI) (*poem.Poem, string, error) {
	now := time.Now()

	log.Println("📜 Fetching a random poem...")
	p := fetchRandomPoem()

	// yamlPoem, err := yaml.Marshal(p)
	// if err != nil {
	// 	return nil, "", fmt.Errorf("failed to generate yaml for poem: %w", err)
	// }
	// os.Stdout.Write(yamlPoem)

	log.Printf("🧹 Making poem %s - %s to Reading Script...", p.Title, p.Author)
	if err := aiFix.CleanupContent(ctx, p); err != nil {
		return nil, "", fmt.Errorf("CleanupContent failed: %w", err)
	}

	if err := aiFix.FillReadingScript(ctx, p); err != nil {
		return nil, "", fmt.Errorf("FixContentForTTS failed: %w", err)
	}

	log.Println("📜 Poem Script for Reading:")
	log.Println(p.ReadingScript)
	log.Println("⚙️ Generating TTS...")

	// use unix nano to guarantee unique file names across tight generation loops
	wavName := fmt.Sprintf("poem_%d.wav", now.UnixNano())
	wavFile, err := os.Create(wavName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create %s: %w", wavName, err)
	}

	if err := t.EncodeWavIO(wavFile, p.ReadingScript); err != nil {
		wavFile.Close()
		return nil, "", fmt.Errorf("failed to encode wav: %w", err)
	}

	finalWavName := wavName
	if enticSound {
		log.Println("📞 Applying wirephone effect...")
		if _, err := wavFile.Seek(0, 0); err != nil {
			wavFile.Close()
			return nil, "", fmt.Errorf("failed to seek wav file: %w", err)
		}

		phoneWavName := strings.Replace(wavName, ".wav", ".phone.wav", 1)
		phoneWavFile, err := os.Create(phoneWavName)
		if err != nil {
			wavFile.Close()
			return nil, "", fmt.Errorf("failed to create output wav file: %w", err)
		}

		err = wirephone_sound.MakeAntiquePhone(wavFile, phoneWavFile, addNoise)
		wavFile.Close()
		phoneWavFile.Close()

		if err != nil {
			return nil, "", fmt.Errorf("failed to apply wirephone effect: %w", err)
		}
		os.Remove(wavName) // Remove original wave when wirephone effect applied
		finalWavName = phoneWavName
	} else {
		wavFile.Close()
	}

	log.Printf("Successfully created %s\n\n", finalWavName)
	return p, finalWavName, nil
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
