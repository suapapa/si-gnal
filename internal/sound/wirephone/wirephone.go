package wirephone_sound

import (
	"fmt"
	"io"
	"math"
	"math/rand"

	"github.com/go-audio/wav"
)

type biquad struct {
	b0, b1, b2, a1, a2 float64
	x1, x2, y1, y2     float64
}

func (b *biquad) process(x float64) float64 {
	y := b.b0*x + b.b1*b.x1 + b.b2*b.x2 - b.a1*b.y1 - b.a2*b.y2
	b.x2 = b.x1
	b.x1 = x
	b.y2 = b.y1
	b.y1 = y
	return y
}

func newLPF(f0, fs, q float64) *biquad {
	omega := 2.0 * math.Pi * f0 / fs
	alpha := math.Sin(omega) / (2.0 * q)
	costh := math.Cos(omega)

	a0 := 1.0 + alpha
	return &biquad{
		b0: ((1.0 - costh) / 2.0) / a0,
		b1: (1.0 - costh) / a0,
		b2: ((1.0 - costh) / 2.0) / a0,
		a1: (-2.0 * costh) / a0,
		a2: (1.0 - alpha) / a0,
	}
}

func newHPF(f0, fs, q float64) *biquad {
	omega := 2.0 * math.Pi * f0 / fs
	alpha := math.Sin(omega) / (2.0 * q)
	costh := math.Cos(omega)

	a0 := 1.0 + alpha
	return &biquad{
		b0: ((1.0 + costh) / 2.0) / a0,
		b1: (-(1.0 + costh)) / a0,
		b2: ((1.0 + costh) / 2.0) / a0,
		a1: (-2.0 * costh) / a0,
		a2: (1.0 - alpha) / a0,
	}
}

// MakeAntiquePhone applies an antique wirephone effect to a wav stream.
func MakeAntiquePhone(in io.ReadSeeker, out io.WriteSeeker, addNoise bool) error {
	decoder := wav.NewDecoder(in)
	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return fmt.Errorf("failed to decode wav: %w", err)
	}

	fs := float64(buf.Format.SampleRate)

	// Create filters
	// Typical phone line is 300Hz - 3400Hz
	q := 0.707 // Butterworth

	// If stereo or multi-channel, we need biquads per channel.
	// We'll support any channels by having a slice of biquads per channel.
	numChans := buf.Format.NumChannels

	hpf1 := make([]*biquad, numChans)
	hpf2 := make([]*biquad, numChans)
	lpf1 := make([]*biquad, numChans)
	lpf2 := make([]*biquad, numChans)

	for c := 0; c < numChans; c++ {
		hpf1[c] = newHPF(300, fs, q)
		hpf2[c] = newHPF(300, fs, q)
		lpf1[c] = newLPF(3400, fs, q)
		lpf2[c] = newLPF(3400, fs, q)
	}

	// We apply distortion: overdrive/clipping
	// Max int value for normalization based on bit depth
	var maxVal float64
	if buf.SourceBitDepth == 16 {
		maxVal = 32767.0
	} else if buf.SourceBitDepth == 8 {
		maxVal = 127.0 // unsigned normally, but go-audio/wav converts to IntBuffer?
		// Wait, 8 bit is unsigned, but go-audio makes it -128 to 127?
		// For safety, let's just go with float max conversion later.
	} else if buf.SourceBitDepth == 24 {
		maxVal = 8388607.0
	} else if buf.SourceBitDepth == 32 {
		maxVal = 2147483647.0
	} else {
		// Default to 16 bit
		maxVal = 32767.0
		buf.SourceBitDepth = 16
	}

	for i := 0; i < len(buf.Data); i += numChans {
		for c := 0; c < numChans; c++ {
			smp := buf.Data[i+c]

			// Normalize
			val := float64(smp) / maxVal

			// Filter
			val = hpf1[c].process(val)
			val = hpf2[c].process(val)
			val = lpf1[c].process(val)
			val = lpf2[c].process(val)

			// Overdrive / Distortion
			// A simple waveshaper: f(x) = (2/pi) * atan(k*x)
			gain := 8.0
			val = (2.0 / math.Pi) * math.Atan(gain*val)

			// Soft clipping
			if val > 1.0 {
				val = 1.0
			} else if val < -1.0 {
				val = -1.0
			}

			if addNoise {
				// Add some noise
				// Telephone lines have a slight hum and static noise
				noise := (rand.Float64()*2.0 - 1.0) * 0.015
				val += noise
			}

			// Output limiting just in case
			if val > 1.0 {
				val = 1.0
			} else if val < -1.0 {
				val = -1.0
			}

			// Denormalize
			buf.Data[i+c] = int(val * maxVal)
		}
	}

	encoder := wav.NewEncoder(out, buf.Format.SampleRate, buf.SourceBitDepth, buf.Format.NumChannels, 1)
	if err := encoder.Write(buf); err != nil {
		return fmt.Errorf("failed to write wav: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to close encoder: %w", err)
	}

	return nil
}
