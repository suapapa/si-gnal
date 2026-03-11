package dtmf_test

import (
	"bytes"
	"testing"

	"github.com/suapapa/si-gnal/internal/sound/dtmf"
)

func TestGenerateWav(t *testing.T) {
	var buf bytes.Buffer
	input := "1004#"

	err := dtmf.GenerateWav(input, &buf)
	if err != nil {
		t.Fatalf("GenerateWav failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatalf("Expected non-empty buffer")
	}

	// 44 bytes header + tone data
	if buf.Len() <= 44 {
		t.Fatalf("Expected buffer length to be greater than 44 bytes, got %d", buf.Len())
	}

	// Check standard WAV header signature
	data := buf.Bytes()
	if string(data[0:4]) != "RIFF" {
		t.Errorf("Missing RIFF header")
	}
	if string(data[8:12]) != "WAVE" {
		t.Errorf("Missing WAVE format")
	}
}
