package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
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
	useMemory  bool

	supertonicOnnxDir    string
	supertonicVoiceStyle string
)

type Server struct {
	playMu             sync.Mutex
	isPlaying          bool
	playCancel         context.CancelFunc
	currentPlayingFile string
	queueMu            sync.Mutex
	poemQueue          []PlayJob
}

func NewServer(batch int) *Server {
	return &Server{
		poemQueue: make([]PlayJob, 0, batch),
	}
}

type PlayJob struct {
	WavName string     `json:"wavName"`
	WavData []byte     `json:"-"`
	Poem    *poem.Poem `json:"poem"`
}

func main() {
	flag.IntVar(&batch, "b", 5, "batch count for pre-generated wav files")
	flag.BoolVar(&enticSound, "e", false, "apply wire-phone effect on output")
	flag.BoolVar(&addNoise, "n", false, "add noise")
	flag.StringVar(&ttsEngine, "t", "supertonic", "tts engine (supertonic, htgo)")
	flag.StringVar(&supertonicOnnxDir, "s", "assets/supertonic2/onnx", "supertonic onnx directory")
	flag.StringVar(&supertonicVoiceStyle, "v", "assets/supertonic2/voice_styles/F5.json", "supertonic voice style")
	flag.BoolVar(&playOutput, "p", false, "play the output wav file directly to speaker (legacy)")
	flag.BoolVar(&useMemory, "m", false, "store generated wav files in memory instead of files")
	flag.Parse()

	srv := NewServer(batch)

	// init engines
	var t tts.TTS
	var err error

	switch ttsEngine {
	case "supertonic":
		ttsParams := supertonic.NewDefaultParameters()
		ttsParams.TotalStep = 32
		ttsParams.Speed = 0.85
		ttsParams.SilenceDuration = 1.2
		ttsParams.ONNXDir = supertonicOnnxDir
		ttsParams.VoiceStyles = []string{supertonicVoiceStyle}
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

	defer aiFix.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() {
		log.Println("🧹 대기 큐 정리중...")
		srv.queueMu.Lock()
		for _, job := range srv.poemQueue {
			if len(job.WavData) == 0 {
				os.Remove(job.WavName)
			}
		}
		srv.poemQueue = nil
		srv.queueMu.Unlock()
		log.Println("👋 ByeBye")
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			srv.queueMu.Lock()
			if len(srv.poemQueue) >= batch {
				srv.queueMu.Unlock()
				time.Sleep(1 * time.Second)
				continue
			}
			srv.queueMu.Unlock()

			p, wavFile, wavData, err := generateOneWav(ctx, t, aiFix)
			if err != nil {
				log.Printf("failed to generate wav: %v", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Minute):
				}
				continue
			}

			srv.queueMu.Lock()
			// check ctx again before adding
			select {
			case <-ctx.Done():
				if !useMemory {
					os.Remove(wavFile)
				}
				srv.queueMu.Unlock()
				return
			default:
				srv.poemQueue = append(srv.poemQueue, PlayJob{
					WavName: wavFile,
					WavData: wavData,
					Poem:    p,
				})
				srv.queueMu.Unlock()
			}
		}
	}()

	r := gin.Default()

	r.POST("/api/stop", srv.handleStop)

	r.GET("/api/poem", srv.handleGetPoems)
	r.GET("/api/poem/head", srv.handleGetPoemHead)
	r.GET("/api/poem/pop", srv.handleGetPoemPop)

	port := ":8080"
	httpSrv := &http.Server{
		Addr:    port,
		Handler: r,
	}

	log.Printf("🌐 Starting web server on %s...", port)
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exiting")
}

func generateOneWav(ctx context.Context, t tts.TTS, aiFix *ai.AI) (*poem.Poem, string, []byte, error) {
	now := time.Now()

	log.Println("📜 Fetching a random poem...")
	p := fetchRandomPoem()

	log.Printf("🧹 Making poem %s - %s to Reading Script...", p.Title, p.Author)
	if err := aiFix.CleanupContent(ctx, p); err != nil {
		return nil, "", nil, fmt.Errorf("CleanupContent failed: %w", err)
	}

	if err := aiFix.FillReadingScript(ctx, p); err != nil {
		return nil, "", nil, fmt.Errorf("FixContentForTTS failed: %w", err)
	}

	log.Println("📜 Poem Script for Reading:")
	log.Println(p.ReadingScript)
	log.Println("⚙️ Generating TTS...")

	// use unix nano to guarantee unique file names across tight generation loops
	wavName := fmt.Sprintf("poem_%d.wav", now.UnixNano())

	var ws io.WriteSeeker
	var memW *MemoryReadWriteSeeker

	if useMemory {
		memW = &MemoryReadWriteSeeker{}
		ws = memW
	} else {
		wavFile, err := os.Create(wavName)
		if err != nil {
			return nil, "", nil, fmt.Errorf("failed to create %s: %w", wavName, err)
		}
		ws = wavFile
		defer wavFile.Close()
	}

	if err := t.EncodeWavIO(ws, p.ReadingScript); err != nil {
		return nil, "", nil, fmt.Errorf("failed to encode wav: %w", err)
	}

	finalWavName := wavName
	if enticSound {
		log.Println("📞 Applying wirephone effect...")
		var rs io.ReadSeeker
		if useMemory {
			rs = memW
		} else {
			f, err := os.Open(wavName)
			if err != nil {
				return nil, "", nil, fmt.Errorf("failed to open wav for effect: %w", err)
			}
			defer f.Close()
			rs = f
		}

		if _, err := rs.Seek(0, 0); err != nil {
			return nil, "", nil, fmt.Errorf("failed to seek wav: %w", err)
		}

		phoneWavName := strings.Replace(wavName, ".wav", ".phone.wav", 1)
		var outWs io.WriteSeeker
		var outMemW *MemoryReadWriteSeeker

		if useMemory {
			outMemW = &MemoryReadWriteSeeker{}
			outWs = outMemW
		} else {
			phoneWavFile, err := os.Create(phoneWavName)
			if err != nil {
				return nil, "", nil, fmt.Errorf("failed to create output wav file: %w", err)
			}
			defer phoneWavFile.Close()
			outWs = phoneWavFile
		}

		err := wirephone_sound.MakeAntiquePhone(rs, outWs, addNoise)
		if err != nil {
			return nil, "", nil, fmt.Errorf("failed to apply wirephone effect: %w", err)
		}

		if useMemory {
			memW = outMemW
		} else {
			os.Remove(wavName) // Remove original wave when wirephone effect applied
		}
		finalWavName = phoneWavName
	}

	var wavData []byte
	if useMemory {
		wavData = memW.Bytes()
		wavName = finalWavName
	} else {
		wavName = finalWavName
	}

	log.Printf("Successfully created %s (memory: %v)\n\n", finalWavName, useMemory)
	return p, wavName, wavData, nil
}

type MemoryReadWriteSeeker struct {
	buf []byte
	pos int64
}

func (m *MemoryReadWriteSeeker) Write(p []byte) (n int, err error) {
	if m.pos == int64(len(m.buf)) {
		m.buf = append(m.buf, p...)
		m.pos += int64(len(p))
		return len(p), nil
	}
	if m.pos+int64(len(p)) > int64(len(m.buf)) {
		newBuf := make([]byte, m.pos+int64(len(p)))
		copy(newBuf, m.buf)
		m.buf = newBuf
	}
	n = copy(m.buf[m.pos:], p)
	m.pos += int64(n)
	return n, nil
}

func (m *MemoryReadWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = m.pos + offset
	case io.SeekEnd:
		newPos = int64(len(m.buf)) + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}
	if newPos < 0 {
		return 0, fmt.Errorf("negative position")
	}
	m.pos = newPos
	return m.pos, nil
}

func (m *MemoryReadWriteSeeker) Read(p []byte) (n int, err error) {
	if m.pos >= int64(len(m.buf)) {
		return 0, io.EOF
	}
	n = copy(p, m.buf[m.pos:])
	m.pos += int64(n)
	return n, nil
}

func (m *MemoryReadWriteSeeker) Bytes() []byte {
	return m.buf
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
