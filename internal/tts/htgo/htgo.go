package htgo

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/go-mp3"
	htgotts "github.com/hegedustibor/htgo-tts"
	"github.com/hegedustibor/htgo-tts/handlers"
)

type TTS struct {
	speech  *htgotts.Speech
	tempDir string
}

func NewTTS(lang string) (*TTS, error) {
	tempDir, err := os.MkdirTemp("", "htgo_tts")
	if err != nil {
		return nil, err
	}
	speech := &htgotts.Speech{
		Folder:   tempDir,
		Language: lang,
		Handler:  &handlers.Native{}, // Not actively used but avoids nil pointer
	}
	return &TTS{
		speech:  speech,
		tempDir: tempDir,
	}, nil
}

func (t *TTS) Close() {
	if t.tempDir != "" {
		os.RemoveAll(t.tempDir)
	}
}

func (t *TTS) EncodeWavIO(w io.WriteSeeker, text string) error {
	// htgo-tts generates file with hash name when using Speak, but CreateSpeechBuff asks for a name.
	fileName := fmt.Sprintf("temp_%d", time.Now().UnixNano())
	_, err := t.speech.CreateSpeechBuff(text, fileName)
	if err != nil {
		return err
	}

	// htgotts package has a bug: CreateSpeechBuff drains the buffer when saving to disk,
	// so the returned io.Reader is empty. We must read the saved file instead.
	mp3Path := filepath.Join(t.tempDir, fileName+".mp3")
	mp3File, err := os.Open(mp3Path)
	if err != nil {
		return err
	}
	defer func() {
		mp3File.Close()
		os.Remove(mp3Path) // clean up immediately
	}()

	decoder, err := mp3.NewDecoder(mp3File)
	if err != nil {
		return err
	}

	dataSize := uint32(decoder.Length())
	sampleRate := uint32(decoder.SampleRate())
	numChannels := uint16(2) // go-mp3 always outputs 2 channels
	bitDepth := uint16(16)   // go-mp3 always outputs 16-bit
	byteRate := sampleRate * uint32(numChannels) * uint32(bitDepth) / 8
	blockAlign := numChannels * bitDepth / 8

	// RIFF chunk
	if _, err := w.Write([]byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(36+dataSize)); err != nil {
		return err
	}
	if _, err := w.Write([]byte("WAVE")); err != nil {
		return err
	}

	// fmt subchunk
	if _, err := w.Write([]byte("fmt ")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(1)); err != nil { // PCM
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, numChannels); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, sampleRate); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, byteRate); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, blockAlign); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, bitDepth); err != nil {
		return err
	}

	// data subchunk
	if _, err := w.Write([]byte("data")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, dataSize); err != nil {
		return err
	}

	// copy decoded mp3 PCM data to wav
	_, err = io.Copy(w, decoder)
	return err
}
