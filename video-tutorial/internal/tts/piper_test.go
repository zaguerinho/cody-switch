package tts

import (
	"math"
	"testing"
)

func TestEstimateWordTimestamps(t *testing.T) {
	t.Run("distributes duration proportionally by character count", func(t *testing.T) {
		words := estimateWordTimestamps("Hello world", 2.0)
		if len(words) != 2 {
			t.Fatalf("got %d words, want 2", len(words))
		}
		if words[0].Word != "Hello" {
			t.Errorf("word[0] = %q, want Hello", words[0].Word)
		}
		if words[1].Word != "world" {
			t.Errorf("word[1] = %q, want world", words[1].Word)
		}
		// "Hello" = 5 chars, "world" = 5 chars → equal split
		if math.Abs(words[0].End-words[0].Start-1.0) > 0.01 {
			t.Errorf("word[0] duration = %v, want ~1.0", words[0].End-words[0].Start)
		}
		if math.Abs(words[1].End-words[1].Start-1.0) > 0.01 {
			t.Errorf("word[1] duration = %v, want ~1.0", words[1].End-words[1].Start)
		}
		// First word starts at 0
		if words[0].Start != 0 {
			t.Errorf("word[0].Start = %v, want 0", words[0].Start)
		}
	})

	t.Run("longer words get more time", func(t *testing.T) {
		words := estimateWordTimestamps("I extraordinary", 3.0)
		if len(words) != 2 {
			t.Fatalf("got %d words, want 2", len(words))
		}
		// "I" = 1 char, "extraordinary" = 13 chars
		// I should get 1/14 * 3.0 ≈ 0.214
		// extraordinary should get 13/14 * 3.0 ≈ 2.786
		iDuration := words[0].End - words[0].Start
		exDuration := words[1].End - words[1].Start
		if exDuration <= iDuration {
			t.Errorf("longer word should get more time: I=%v, extraordinary=%v", iDuration, exDuration)
		}
	})

	t.Run("handles empty text", func(t *testing.T) {
		words := estimateWordTimestamps("", 1.0)
		if len(words) != 0 {
			t.Fatalf("got %d words, want 0", len(words))
		}
	})

	t.Run("handles single word", func(t *testing.T) {
		words := estimateWordTimestamps("Hello", 0.5)
		if len(words) != 1 {
			t.Fatalf("got %d words, want 1", len(words))
		}
		if words[0].Start != 0 {
			t.Errorf("start = %v, want 0", words[0].Start)
		}
		if math.Abs(words[0].End-0.5) > 0.01 {
			t.Errorf("end = %v, want 0.5", words[0].End)
		}
	})

	t.Run("timestamps are monotonic", func(t *testing.T) {
		words := estimateWordTimestamps("one two three four five", 5.0)
		for i := 1; i < len(words); i++ {
			if words[i].Start < words[i-1].End {
				t.Errorf("word[%d].Start (%v) < word[%d].End (%v)", i, words[i].Start, i-1, words[i-1].End)
			}
		}
	})

	t.Run("total duration equals input", func(t *testing.T) {
		words := estimateWordTimestamps("hello beautiful world", 3.0)
		lastEnd := words[len(words)-1].End
		if math.Abs(lastEnd-3.0) > 0.01 {
			t.Errorf("last word ends at %v, want 3.0", lastEnd)
		}
	})
}

func TestPcmToWAV(t *testing.T) {
	// Create minimal PCM data (2 samples = 4 bytes at 16-bit)
	pcm := []byte{0x00, 0x01, 0x00, 0x02}
	wav := pcmToWAV(pcm, 22050, 1, 16)

	// Check RIFF header
	if string(wav[0:4]) != "RIFF" {
		t.Errorf("missing RIFF header")
	}
	if string(wav[8:12]) != "WAVE" {
		t.Errorf("missing WAVE marker")
	}
	if string(wav[12:16]) != "fmt " {
		t.Errorf("missing fmt chunk")
	}
	if string(wav[36:40]) != "data" {
		t.Errorf("missing data chunk")
	}

	// Total size should be 44 (header) + 4 (data)
	if len(wav) != 48 {
		t.Errorf("WAV size = %d, want 48", len(wav))
	}
}

func TestPiperName(t *testing.T) {
	p := &Piper{}
	if p.Name() != "Piper" {
		t.Errorf("Name() = %q, want Piper", p.Name())
	}
}
