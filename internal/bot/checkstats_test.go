package bot

import (
	"testing"
	"time"
)

func TestComputeStreak(t *testing.T) {
	today, err := time.Parse("2006-01-02", "2026-09-16")
	if err != nil {
		t.Fatalf("time.Parse() error = %v", err)
	}

	tests := []struct {
		name string
		done map[string]bool
		want int
	}{
		{
			name: "danas i tri dana unazad odrađeno",
			done: map[string]bool{
				"2026-09-16": true,
				"2026-09-15": true,
				"2026-09-14": true,
				"2026-09-13": true,
			},
			want: 4,
		},
		{
			name: "danas još nije odrađeno, ali juče jeste - niz se ne prekida odmah",
			done: map[string]bool{
				"2026-09-15": true,
				"2026-09-14": true,
			},
			want: 2,
		},
		{
			name: "rupa pre dva dana prekida niz",
			done: map[string]bool{
				"2026-09-16": true,
				"2026-09-14": true, // rupa na 2026-09-15
			},
			want: 1,
		},
		{
			name: "nema nijednog zapisa",
			done: map[string]bool{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeStreak(tt.done, today)
			if got != tt.want {
				t.Errorf("computeStreak() = %d, want %d", got, tt.want)
			}
		})
	}
}
