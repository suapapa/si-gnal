package dtmf

import (
	"encoding/binary"
	"io"
	"math"
)

const (
	sampleRate = 8000
	duration   = 0.2  // 200ms tone duration
	pause      = 0.05 // 50ms pause between tones
)

var dtmfFrequencies = map[rune][2]float64{
	'1': {697, 1209},
	'2': {697, 1336},
	'3': {697, 1477},
	'A': {697, 1633},
	'a': {697, 1633},
	'4': {770, 1209},
	'5': {770, 1336},
	'6': {770, 1477},
	'B': {770, 1633},
	'b': {770, 1633},
	'7': {852, 1209},
	'8': {852, 1336},
	'9': {852, 1477},
	'C': {852, 1633},
	'c': {852, 1633},
	'*': {941, 1209},
	'0': {941, 1336},
	'#': {941, 1477},
	'D': {941, 1633},
	'd': {941, 1633},
}

// GenerateWav writes DTMF tones encoded as a WAV file to the provided io.Writer
func GenerateWav(input string, w io.Writer) error {
	var validChars []rune
	for _, r := range input {
		if _, ok := dtmfFrequencies[r]; ok {
			validChars = append(validChars, r)
		}
	}

	numSamplesPerTone := int(sampleRate * duration)
	numSamplesPerPause := int(sampleRate * pause)
	numTones := len(validChars)

	if numTones == 0 {
		return writeWavHeader(w, 0)
	}

	totalSamples := numSamplesPerTone*numTones + numSamplesPerPause*(numTones-1)

	err := writeWavHeader(w, totalSamples)
	if err != nil {
		return err
	}

	for i, r := range validChars {
		freqs := dtmfFrequencies[r]

		// generate tone
		for j := 0; j < numSamplesPerTone; j++ {
			t := float64(j) / float64(sampleRate)
			// mixing two sine waves
			val1 := math.Sin(2 * math.Pi * freqs[0] * t)
			val2 := math.Sin(2 * math.Pi * freqs[1] * t)

			// Normalize to -1.0 to 1.0, then scale to 16-bit int
			sample := (val1 + val2) / 2.0
			intSample := int16(sample * 32767)

			if err := binary.Write(w, binary.LittleEndian, intSample); err != nil {
				return err
			}
		}

		// generate pause
		if i < numTones-1 {
			for j := 0; j < numSamplesPerPause; j++ {
				if err := binary.Write(w, binary.LittleEndian, int16(0)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func writeWavHeader(w io.Writer, numSamples int) error {
	numChannels := 1
	bitsPerSample := 16
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8
	dataSize := numSamples * numChannels * bitsPerSample / 8
	chunkSize := 36 + dataSize

	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(chunkSize))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16) // Subchunk1Size (16 for PCM)
	binary.LittleEndian.PutUint16(header[20:22], 1)  // AudioFormat (1 for PCM)
	binary.LittleEndian.PutUint16(header[22:24], uint16(numChannels))
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:32], uint32(byteRate))
	binary.LittleEndian.PutUint16(header[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitsPerSample))
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize))

	_, err := w.Write(header)
	return err
}
