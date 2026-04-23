package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"github.com/zaguerinho/claude-switch/video-tutorial/internal/config"
)

// Piper implements the Provider interface using the local Piper TTS engine.
// Piper outputs WAV audio and does not provide native word-level timestamps,
// so timestamps are estimated by distributing audio duration proportionally
// across words by character count.
type Piper struct {
	cfg *config.Config
}

// NewPiper creates a Piper provider using the given config.
func NewPiper(cfg *config.Config) *Piper {
	return &Piper{cfg: cfg}
}

// Name returns "Piper".
func (p *Piper) Name() string { return "Piper" }

// SynthesizeSegment synthesizes text using the local Piper binary.
// Returns audio as base64-encoded WAV with estimated word timestamps.
func (p *Piper) SynthesizeSegment(ctx context.Context, text string) (*SynthResult, error) {
	// Build piper command — try `piper` binary first, fall back to `python3 -m piper`
	piperBin, piperArgs := resolvePiperCommand()
	args := append(piperArgs, "--output-raw")
	if p.cfg.PiperModel != "" {
		args = append(args, "--model", p.cfg.PiperModel)
	}
	if p.cfg.PiperDataDir != "" {
		args = append(args, "--data-dir", p.cfg.PiperDataDir)
	}

	cmd := exec.CommandContext(ctx, piperBin, args...)
	cmd.Stdin = strings.NewReader(text)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("piper synthesis failed: %w\nstderr: %s", err, stderr.String())
	}

	rawPCM := stdout.Bytes()
	if len(rawPCM) == 0 {
		return nil, fmt.Errorf("piper produced no audio output")
	}

	// Piper --output-raw produces 16-bit mono PCM at the model's sample rate.
	// Default is 22050 Hz for most Piper models.
	sampleRate := p.cfg.PiperSampleRate
	if sampleRate == 0 {
		sampleRate = 22050
	}

	// Wrap raw PCM in a WAV container so the browser's decodeAudioData can handle it.
	wavData := pcmToWAV(rawPCM, sampleRate, 1, 16)

	// Calculate duration from audio data.
	numSamples := len(rawPCM) / 2 // 16-bit = 2 bytes per sample
	duration := float64(numSamples) / float64(sampleRate)

	// Encode to base64.
	audioBase64 := base64.StdEncoding.EncodeToString(wavData)

	// Estimate word timestamps proportionally by character count.
	words := estimateWordTimestamps(text, duration)

	return &SynthResult{
		AudioBase64:    audioBase64,
		WordTimestamps: words,
		Duration:       duration,
	}, nil
}

// estimateWordTimestamps splits text into words and distributes the total
// audio duration proportionally based on character count. This gives longer
// words more time, which is a reasonable approximation when real timestamps
// aren't available.
func estimateWordTimestamps(text string, totalDuration float64) []WordTimestamp {
	// Split into words, preserving only non-empty words.
	rawWords := strings.Fields(text)
	if len(rawWords) == 0 {
		return nil
	}

	// Calculate total character weight (use rune count for Unicode support).
	totalChars := 0
	wordLens := make([]int, len(rawWords))
	for i, w := range rawWords {
		n := 0
		for _, r := range w {
			if !unicode.IsSpace(r) {
				n++
			}
		}
		if n == 0 {
			n = 1 // minimum weight
		}
		wordLens[i] = n
		totalChars += n
	}

	// Distribute duration proportionally.
	var timestamps []WordTimestamp
	offset := 0.0
	for i, w := range rawWords {
		wordDuration := totalDuration * float64(wordLens[i]) / float64(totalChars)
		timestamps = append(timestamps, WordTimestamp{
			Word:  w,
			Start: offset,
			End:   offset + wordDuration,
		})
		offset += wordDuration
	}

	return timestamps
}

// pcmToWAV wraps raw PCM data in a standard WAV header.
func pcmToWAV(pcmData []byte, sampleRate, numChannels, bitsPerSample int) []byte {
	dataSize := len(pcmData)
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8

	var buf bytes.Buffer

	// RIFF header
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+dataSize)) // file size - 8
	buf.WriteString("WAVE")

	// fmt sub-chunk
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))                // sub-chunk size
	binary.Write(&buf, binary.LittleEndian, uint16(1))                 // PCM format
	binary.Write(&buf, binary.LittleEndian, uint16(numChannels))       // channels
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))        // sample rate
	binary.Write(&buf, binary.LittleEndian, uint32(byteRate))          // byte rate
	binary.Write(&buf, binary.LittleEndian, uint16(blockAlign))        // block align
	binary.Write(&buf, binary.LittleEndian, uint16(bitsPerSample))     // bits per sample

	// data sub-chunk
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(dataSize))
	buf.Write(pcmData)

	return buf.Bytes()
}

// resolvePiperCommand returns the binary and base args to invoke Piper.
// Tries `piper` on PATH first, then `python3 -m piper`.
func resolvePiperCommand() (string, []string) {
	if _, err := exec.LookPath("piper"); err == nil {
		return "piper", nil
	}
	// Fall back to Python module
	return "python3", []string{"-m", "piper"}
}

// IsPiperInstalled checks if Piper is available (either as a binary or Python module).
func IsPiperInstalled() bool {
	if _, err := exec.LookPath("piper"); err == nil {
		return true
	}
	// Check if python3 -m piper works
	cmd := exec.Command("python3", "-m", "piper", "--help")
	return cmd.Run() == nil
}

// InstallPiperHint returns a user-friendly message about how to install Piper.
func InstallPiperHint() string {
	return `Piper TTS is not installed. Install it with:

  pip install piper-tts

Or download the binary from:
  https://github.com/rhasspy/piper/releases

After installing, Piper will auto-download voice models on first use.
To specify a model: --piper-model en_US-lessac-medium`
}

// EnsurePiperModel checks if the specified model is available, and prints
// a message that Piper will auto-download it if needed.
func EnsurePiperModel(model string) {
	if model != "" {
		fmt.Fprintf(os.Stderr, "Using Piper model: %s (will auto-download if needed)\n", model)
	} else {
		fmt.Fprintf(os.Stderr, "Using Piper default model (will auto-download if needed)\n")
	}
}
