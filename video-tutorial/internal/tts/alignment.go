// Package tts provides ElevenLabs TTS synthesis with word-level timestamps.
package tts

// Alignment holds the character-level timing data returned by ElevenLabs.
type Alignment struct {
	Characters []string  `json:"characters"`
	Starts     []float64 `json:"character_start_times_seconds"`
	Ends       []float64 `json:"character_end_times_seconds"`
}

// WordTimestamp represents a single word with its start and end time in seconds.
type WordTimestamp struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// CharacterTimestampsToWords converts character-level alignment data from
// ElevenLabs into word-level timestamps. Groups consecutive non-whitespace
// characters into words. Direct port of characterTimestampsToWords from
// elevenlabs.js.
func CharacterTimestampsToWords(a Alignment) []WordTimestamp {
	var words []WordTimestamp
	var currentWord []byte
	var wordStart, wordEnd float64
	wordStart = -1

	for i := 0; i < len(a.Characters); i++ {
		ch := a.Characters[i]
		if ch == " " || ch == "\n" || ch == "\t" {
			if len(currentWord) > 0 {
				words = append(words, WordTimestamp{
					Word:  string(currentWord),
					Start: wordStart,
					End:   wordEnd,
				})
				currentWord = currentWord[:0]
				wordStart = -1
			}
		} else {
			if wordStart < 0 {
				wordStart = a.Starts[i]
			}
			currentWord = append(currentWord, ch...)
			wordEnd = a.Ends[i]
		}
	}

	// Flush last word
	if len(currentWord) > 0 {
		words = append(words, WordTimestamp{
			Word:  string(currentWord),
			Start: wordStart,
			End:   wordEnd,
		})
	}

	return words
}
