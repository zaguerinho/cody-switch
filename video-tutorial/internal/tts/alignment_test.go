package tts

import (
	"testing"
)

func TestCharacterTimestampsToWords(t *testing.T) {
	tests := []struct {
		name     string
		align    Alignment
		wantLen  int
		wantWord []string
		checks   func(t *testing.T, words []WordTimestamp)
	}{
		{
			name: "groups characters into words",
			align: Alignment{
				Characters: []string{"H", "e", "l", "l", "o", " ", "w", "o", "r", "l", "d"},
				Starts:     []float64{0, 0.1, 0.15, 0.2, 0.25, 0.3, 0.35, 0.4, 0.45, 0.5, 0.55},
				Ends:       []float64{0.1, 0.15, 0.2, 0.25, 0.3, 0.35, 0.4, 0.45, 0.5, 0.55, 0.6},
			},
			wantLen:  2,
			wantWord: []string{"Hello", "world"},
			checks: func(t *testing.T, words []WordTimestamp) {
				if words[0].Start != 0 {
					t.Errorf("word[0].Start = %v, want 0", words[0].Start)
				}
				if words[0].End != 0.3 {
					t.Errorf("word[0].End = %v, want 0.3", words[0].End)
				}
				if words[1].Start != 0.35 {
					t.Errorf("word[1].Start = %v, want 0.35", words[1].Start)
				}
				if words[1].End != 0.6 {
					t.Errorf("word[1].End = %v, want 0.6", words[1].End)
				}
			},
		},
		{
			name: "handles single word",
			align: Alignment{
				Characters: []string{"O", "K"},
				Starts:     []float64{0, 0.1},
				Ends:       []float64{0.1, 0.2},
			},
			wantLen:  1,
			wantWord: []string{"OK"},
		},
		{
			name: "handles trailing space",
			align: Alignment{
				Characters: []string{"H", "i", " "},
				Starts:     []float64{0, 0.1, 0.2},
				Ends:       []float64{0.1, 0.2, 0.3},
			},
			wantLen:  1,
			wantWord: []string{"Hi"},
		},
		{
			name: "handles multiple spaces between words",
			align: Alignment{
				Characters: []string{"a", " ", " ", "b"},
				Starts:     []float64{0, 0.1, 0.2, 0.3},
				Ends:       []float64{0.1, 0.2, 0.3, 0.4},
			},
			wantLen:  2,
			wantWord: []string{"a", "b"},
		},
		{
			name: "handles empty input",
			align: Alignment{
				Characters: []string{},
				Starts:     []float64{},
				Ends:       []float64{},
			},
			wantLen: 0,
		},
		{
			name: "handles punctuation attached to words",
			align: Alignment{
				Characters: []string{"H", "i", ".", " ", "O", "K", "!"},
				Starts:     []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6},
				Ends:       []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7},
			},
			wantLen:  2,
			wantWord: []string{"Hi.", "OK!"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words := CharacterTimestampsToWords(tt.align)
			if len(words) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(words), tt.wantLen)
			}
			for i, want := range tt.wantWord {
				if words[i].Word != want {
					t.Errorf("word[%d] = %q, want %q", i, words[i].Word, want)
				}
			}
			if tt.checks != nil {
				tt.checks(t, words)
			}
		})
	}
}
