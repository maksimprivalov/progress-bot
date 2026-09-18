package chart

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

type Point struct {
	Date  time.Time
	Value float64
}

func GenerateProgressPNG(title string, points []Point) ([]byte, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("nema podataka za iscrtavanje")
	}

	sorted := make([]Point, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date.Before(sorted[j].Date)
	})

	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = "Date"
	p.Y.Label.Text = "Value"
	p.X.Tick.Marker = dateTicker{points: sorted}

	xys := make(plotter.XYs, len(sorted))
	for i, pt := range sorted {
		xys[i].X = float64(pt.Date.Unix())
		xys[i].Y = pt.Value
	}

	line, scatter, err := plotter.NewLinePoints(xys)
	if err != nil {
		return nil, fmt.Errorf("kreiranje linije grafikona: %w", err)
	}
	p.Add(line, scatter)

	writerTo, err := p.WriterTo(6*vg.Inch, 4*vg.Inch, "png")
	if err != nil {
		return nil, fmt.Errorf("renderovanje grafikona: %w", err)
	}

	var buf bytes.Buffer
	if _, err := writerTo.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("upis PNG podataka u bafer: %w", err)
	}

	return buf.Bytes(), nil
}

type dateTicker struct {
	points []Point
}

func (d dateTicker) Ticks(min, max float64) []plot.Tick {
	ticks := make([]plot.Tick, 0, len(d.points))
	for _, pt := range d.points {
		ticks = append(ticks, plot.Tick{
			Value: float64(pt.Date.Unix()),
			Label: pt.Date.Format("02.01"),
		})
	}
	return ticks
}
