package player

import (
	"os"
	"sync"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/wav"
)

var (
	mu           sync.Mutex
	inited       bool
	initedSample beep.SampleRate
)

// PlayWav plays the wav file at the given filepath through the system's default speaker.
func PlayWav(filepath string) error {
	f, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	streamer, format, err := wav.Decode(f)
	if err != nil {
		return err
	}
	defer streamer.Close()

	mu.Lock()
	if !inited || initedSample != format.SampleRate {
		if inited {
			speaker.Close()
		}
		// Init the speaker with the sample rate and a buffer size of 1/10th of a second
		err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		if err != nil {
			mu.Unlock()
			return err
		}
		inited = true
		initedSample = format.SampleRate
	}
	mu.Unlock()

	// done channel will be triggered after the track is fully played
	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	// wait for it to finish playing
	<-done

	// Add a small sleep to ensure audio buffer finishes flushing
	time.Sleep(100 * time.Millisecond)

	return nil
}
