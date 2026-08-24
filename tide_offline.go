// tide_offline.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"time"
)

type Constituent [3]float64 // amplitude, period, phase_deg

type Config struct {
	Constituents []Constituent `json:"constituents"`
	MeanLevel    float64       `json:"mean_level"`
}

var defaultConfig = Config{
	Constituents: []Constituent{
		{1.2, 12.42, 0.0},
		{0.4, 12.00, 0.0},
		{0.2, 12.66, 0.0},
		{0.1, 23.93, 0.0},
		{0.08, 25.82, 0.0},
	},
	MeanLevel: 0.5,
}

type TideModel struct {
	Constituents []Constituent
	MeanLevel    float64
}

func NewTideModel(cfg Config) *TideModel {
	return &TideModel{
		Constituents: cfg.Constituents,
		MeanLevel:    cfg.MeanLevel,
	}
}

func (t *TideModel) Level(hours float64) float64 {
	level := t.MeanLevel
	for _, c := range t.Constituents {
		amp, period, phase := c[0], c[1], c[2]
		phaseRad := phase * math.Pi / 180.0
		omega := 2 * math.Pi / period
		level += amp * math.Sin(omega*hours+phaseRad)
	}
	return level
}

func (t *TideModel) FindExtrema(start, duration float64) ([]struct{ t, lvl float64 }, []struct{ t, lvl float64 }) {
	step := 0.05
	var times []float64
	var levels []float64
	for t := start; t <= start+duration; t += step {
		times = append(times, t)
		levels = append(levels, t.Level(t))
	}
	var highs, lows []struct{ t, lvl float64 }
	for i := 1; i < len(times)-1; i++ {
		if levels[i] > levels[i-1] && levels[i] > levels[i+1] {
			highs = append(highs, struct{ t, lvl float64 }{times[i], levels[i]})
		} else if levels[i] < levels[i-1] && levels[i] < levels[i+1] {
			lows = append(lows, struct{ t, lvl float64 }{times[i], levels[i]})
		}
	}
	return highs, lows
}

func hoursSinceEpoch(t time.Time) float64 {
	epoch := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	return t.Sub(epoch).Hours()
}

func formatTime(hours float64) string {
	h := int(hours) % 24
	m := int((hours - float64(h)) * 60)
	return fmt.Sprintf("%02d:%02d", h, m)
}

func drawGraph(levels []float64, start, duration float64, width, height int) string {
	if len(levels) == 0 {
		return "No data"
	}
	min, max := levels[0], levels[0]
	for _, v := range levels {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if max-min == 0 {
		return "Flat tide"
	}
	cols := width
	step := duration / float64(cols)
	sampled := make([]float64, cols)
	for i := 0; i < cols; i++ {
		t := start + float64(i)*step
		idx := int((t - start) / step)
		if idx >= len(levels) {
			idx = len(levels) - 1
		}
		sampled[i] = levels[idx]
	}
	minS, maxS := sampled[0], sampled[0]
	for _, v := range sampled {
		if v < minS {
			minS = v
		}
		if v > maxS {
			maxS = v
		}
	}
	rangeS := maxS - minS
	if rangeS == 0 {
		rangeS = 1
	}
	grid := make([][]byte, height)
	for i := range grid {
		grid[i] = make([]byte, cols)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}
	for i, val := range sampled {
		row := int((val - minS) / rangeS * float64(height-1))
		if row < 0 {
			row = 0
		}
		if row >= height {
			row = height - 1
		}
		grid[row][i] = '*'
	}
	var lines []string
	lines = append(lines, "Level (m)")
	for r := height - 1; r >= 0; r-- {
		lvl := minS + (maxS-minS)*float64(r)/float64(height-1)
		line := fmt.Sprintf("%5.2f |", lvl)
		for c := 0; c < cols; c++ {
			line += string(grid[r][c])
		}
		lines = append(lines, line)
	}
	line := "      +" + strings.Repeat("-", cols-1)
	lines = append(lines, line)
	labels := ""
	for h := 0; h <= int(duration); h += 3 {
		pos := int(float64(h) / duration * float64(cols))
		label := fmt.Sprintf("%02d:00", h)
		for len(labels) < pos {
			labels += " "
		}
		labels += label
	}
	lines = append(lines, "       "+labels)
	return strings.Join(lines, "\n")
}

func loadConfig(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func saveConfig(filename string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func main() {
	var (
		dateStr    = flag.String("date", "", "Start date YYYY-MM-DD")
		hours      = flag.Float64("hours", 24.0, "Forecast duration (hours)")
		graphFlag  = flag.Bool("graph", false, "Show ASCII graph")
		listFlag   = flag.Bool("list", false, "Show high/low tides")
		configFile = flag.String("config", "", "JSON config file")
		saveConfig = flag.String("save-config", "", "Save current config to file")
	)
	flag.Parse()

	cfg := defaultConfig
	if *configFile != "" {
		var err error
		cfg, err = loadConfig(*configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
	}

	if *saveConfig != "" {
		if err := saveConfig(*saveConfig, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Config saved to %s\n", *saveConfig)
		return
	}

	var startTime time.Time
	if *dateStr != "" {
		var err error
		startTime, err = time.Parse("2006-01-02", *dateStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid date: %v\n", err)
			os.Exit(1)
		}
	} else {
		now := time.Now().UTC()
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}

	startHours := hoursSinceEpoch(startTime)
	model := NewTideModel(cfg)
	highs, lows := model.FindExtrema(startHours, *hours)

	if *graphFlag {
		levels := []float64{}
		step := 0.1
		for t := startHours; t <= startHours+*hours; t += step {
			levels = append(levels, model.Level(t))
		}
		fmt.Println(drawGraph(levels, startHours, *hours, 50, 20))
	} else {
		fmt.Printf("\n🌊 Offline Tide Calendar\n")
		fmt.Printf("Date: %s – %.0fh forecast\n\n", startTime.Format("2006-01-02 15:04"), *hours)
		fmt.Println("High/Low Tides:")
		events := []struct{ t, lvl float64 }{}
		events = append(events, highs...)
		events = append(events, lows...)
		// sort by time
		for i := 0; i < len(events); i++ {
			for j := i + 1; j < len(events); j++ {
				if events[j].t < events[i].t {
					events[i], events[j] = events[j], events[i]
				}
			}
		}
		for _, e := range events {
			if e.t >= startHours && e.t <= startHours+*hours {
				dt := startTime.Add(time.Duration((e.t-startHours)*3600) * time.Second)
				typ := "High"
				isHigh := false
				for _, h := range highs {
					if math.Abs(h.lvl-e.lvl) < 0.001 {
						isHigh = true
						break
					}
				}
				if !isHigh {
					typ = "Low"
				}
				fmt.Printf("%s  %-4s  %5.2f m\n", dt.Format("2006-01-02 15:04"), typ, e.lvl)
			}
		}
	}
}

import "strings"
