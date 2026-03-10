package tts

import (
	"io"
)

type TTS interface {
	EncodeWavIO(w io.WriteSeeker, text string) error
	Close()
}
