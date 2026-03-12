package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/wav"
)

type headResponse struct {
	Poem struct {
		Title   string `json:"title"`
		Author  string `json:"author"`
		Content string `json:"content"`
	} `json:"poem"`
}

type playerState struct {
	mu           sync.Mutex
	isPlaying    bool
	stopPlayback func()
	initialized  bool
	sampleRate   beep.SampleRate
}

func main() {
	var addr string
	flag.StringVar(&addr, "addr", "http://localhost:8080", "server address")
	flag.Parse()

	if err := keyboard.Open(); err != nil {
		log.Fatal(err)
	}
	defer keyboard.Close()

	fmt.Println("Controls:")
	fmt.Println("  [p] - Play next poem")
	fmt.Println("  [s] - Stop playback")
	fmt.Println("  [q/ESC] - Quit")

	state := &playerState{}

	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			log.Fatal(err)
		}

		if key == keyboard.KeyEsc || char == 'q' {
			break
		}

		switch char {
		case 'p':
			go playPoem(addr, state)
		case 's':
			stopPoem(addr, state)
		}
	}
}

func stopPoem(addr string, state *playerState) {
	state.mu.Lock()
	if state.stopPlayback != nil {
		state.stopPlayback()
	}
	state.mu.Unlock()

	// url := fmt.Sprintf("%s/api/stop", addr)
	// resp, err := http.Post(url, "application/json", nil)
	// if err != nil {
	// 	log.Printf("failed to call stop API: %v", err)
	// 	return
	// }
	// defer resp.Body.Close()
	fmt.Println("\n⏹️ Stopped.")
}

func playPoem(addr string, state *playerState) {
	state.mu.Lock()
	if state.isPlaying {
		state.mu.Unlock()
		fmt.Println("\n⚠️ Already playing!")
		return
	}
	state.isPlaying = true
	state.mu.Unlock()

	defer func() {
		state.mu.Lock()
		state.isPlaying = false
		state.stopPlayback = nil
		state.mu.Unlock()
	}()

	// 1. Fetch poem info from /head
	headUrl := fmt.Sprintf("%s/api/poem/head", addr)
	headResp, err := http.Get(headUrl)
	if err != nil {
		log.Printf("failed to fetch poem head: %v", err)
		return
	}
	defer headResp.Body.Close()

	if headResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(headResp.Body)
		log.Printf("head API returned non-OK status: %s (%s)", headResp.Status, string(body))
		return
	}

	var head headResponse
	if err := json.NewDecoder(headResp.Body).Decode(&head); err != nil {
		log.Printf("failed to decode head response: %v", err)
		return
	}

	fmt.Printf("\n📖 제목: %s\n", head.Poem.Title)
	fmt.Printf("✍️  작가: %s\n\n", head.Poem.Author)
	fmt.Println(head.Poem.Content)

	wg := sync.WaitGroup{}
	var data []byte

	wg.Add(2)
	go func() {
		defer wg.Done()
		// 2. Fetch audio from /pop
		url := fmt.Sprintf("%s/api/poem/pop?play=wav", addr)
		fmt.Printf("Fetching audio from %s...\n", url)

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("failed to call API: %v", err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Printf("API returned non-OK status: %s", resp.Status)
			return
		}

		// Read entire body into memory and close connection
		data, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("failed to read response body: %v", err)
			return
		}
	}()

	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Second)
	}()

	wg.Wait()

	streamer, format, err := wav.Decode(io.NopCloser(bytes.NewReader(data)))
	if err != nil {
		log.Printf("failed to decode wav: %v", err)
		return
	}
	defer streamer.Close()

	// Initialize speaker only once
	state.mu.Lock()
	if !state.initialized {
		err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		if err != nil {
			state.mu.Unlock()
			log.Printf("failed to initialize speaker: %v", err)
			return
		}
		state.initialized = true
		state.sampleRate = format.SampleRate
	}
	state.mu.Unlock()

	var streamerToPlay beep.Streamer = streamer
	if format.SampleRate != state.sampleRate {
		streamerToPlay = beep.Resample(4, format.SampleRate, state.sampleRate, streamer)
	}

	fmt.Println("▶️ Playing audio...")
	done := make(chan bool)
	ctrl := &beep.Ctrl{Streamer: beep.Seq(streamerToPlay, beep.Callback(func() {
		done <- true
	})), Paused: false}

	state.mu.Lock()
	state.stopPlayback = func() {
		speaker.Lock()
		ctrl.Paused = true
		speaker.Unlock()
		select {
		case done <- true:
		default:
		}
	}
	state.mu.Unlock()

	speaker.Play(ctrl)

	<-done
	fmt.Println("Playback finished.")
}
