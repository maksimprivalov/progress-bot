package bot

import (
	"fmt"
	"strings"
	"time"

	"progress-bot/internal/storage"
)

func formatCheckStats(entries []storage.Entry) string {
	if len(entries) == 0 {
		return "Nema još nijednog zapisa."
	}

	done := make(map[string]bool, len(entries))
	for _, e := range entries {
		done[e.EntryDate] = true
	}

	now := time.Now().UTC()
	streak := computeStreak(done, now)
	grid := buildCheckGrid(done, now, 30)

	var sb strings.Builder
	fmt.Fprintf(&sb, "🔥 Trenutni niz: %d dana\n", streak)
	fmt.Fprintf(&sb, "✅ Ukupno odrađeno: %d dana\n\n", len(entries))
	sb.WriteString("Poslednjih 30 dana (najstariji → najnoviji):\n")
	sb.WriteString(grid)

	return sb.String()
}

func computeStreak(done map[string]bool, today time.Time) int {
	day := today
	if !done[day.Format(storage.DateFormat)] {
		day = day.AddDate(0, 0, -1)
	}

	streak := 0
	for done[day.Format(storage.DateFormat)] {
		streak++
		day = day.AddDate(0, 0, -1)
	}
	return streak
}

func buildCheckGrid(done map[string]bool, today time.Time, days int) string {
	const perRow = 10

	start := today.AddDate(0, 0, -(days - 1))
	var sb strings.Builder
	for i := range days {
		day := start.AddDate(0, 0, i)
		if done[day.Format(storage.DateFormat)] {
			sb.WriteString("✅")
		} else {
			sb.WriteString("⬜")
		}
		if (i+1)%perRow == 0 && i != days-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
