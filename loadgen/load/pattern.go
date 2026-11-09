package load

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Pattern struct {
	Name     string
	Peak     float64
	Duration time.Duration
	Phases   []Phase
}

func ParsePeak(value string) (float64, error) {
	peak, err := strconv.ParseFloat(strings.TrimSuffix(value, "/s"), 64)
	if err != nil || !validLevel(peak) || peak == 0 {
		return 0, fmt.Errorf("--peak must be a positive finite iterations/s rate <= 1e9, for example 120/s or 120")
	}
	return peak, nil
}

func Compile(name string, peak float64, duration time.Duration, params map[string]string) (*Pattern, error) {
	var valid []string
	switch name {
	case "steady":
	case "ramp-hold-ramp":
		valid = []string{"up", "down"}
	case "sweep":
		valid = []string{"from", "to", "steps"}
	case "spike-train":
		valid = []string{"base", "every", "for"}
	default:
		return nil, fmt.Errorf("unknown load pattern %q; valid patterns: steady, ramp-hold-ramp, sweep, spike-train", name)
	}
	for key := range params {
		if !slices.Contains(valid, key) {
			choices := strings.Join(valid, ", ")
			if choices == "" {
				choices = "none"
			}
			return nil, fmt.Errorf("unknown parameter %q for %s; valid parameters: %s", key, name, choices)
		}
	}
	if duration <= 0 {
		return nil, fmt.Errorf("--load requires a positive --duration")
	}
	if !validLevel(peak) || peak == 0 {
		return nil, fmt.Errorf("--load requires a positive finite --peak <= 1e9 iterations/s")
	}
	value := func(key, fallback string) string {
		if v, ok := params[key]; ok {
			return v
		}
		return fallback
	}
	p := &Pattern{Name: name, Peak: peak, Duration: duration}
	add := func(name string, duration time.Duration, from, to float64) {
		if duration > 0 {
			p.Phases = append(p.Phases, Phase{name, duration, from * peak, to * peak})
		}
	}
	switch name {
	case "steady":
		add("hold", duration, 1, 1)
	case "ramp-hold-ramp":
		up, err := parseDuration("up", value("up", "10%"), duration)
		if err != nil {
			return nil, err
		}
		down, err := parseDuration("down", value("down", "10%"), duration)
		if err != nil {
			return nil, err
		}
		if up > duration || down > duration-up {
			return nil, fmt.Errorf("up and down durations exceed --duration")
		}
		add("up", up, 0, 1)
		add("hold", duration-up-down, 1, 1)
		add("down", down, 1, 0)
	case "sweep":
		from, err := parseFraction("from", value("from", "25%"))
		if err != nil {
			return nil, err
		}
		to, err := parseFraction("to", value("to", "100%"))
		if err != nil {
			return nil, err
		}
		steps, err := strconv.Atoi(value("steps", "4"))
		if err != nil || steps < 2 || steps > MaxPhases || time.Duration(steps) > duration {
			return nil, fmt.Errorf("steps must be an integer from 2 to %d with at least 1ns per step", MaxPhases)
		}
		stepDuration := duration / time.Duration(steps)
		for i := range steps {
			level := from + (to-from)*float64(i)/float64(steps-1)
			d := stepDuration
			if i == steps-1 {
				d = duration - time.Duration(i)*stepDuration
			}
			add(fmt.Sprintf("step-%d", i+1), d, level, level)
		}
	case "spike-train":
		base, err := parseFraction("base", value("base", "40%"))
		if err != nil {
			return nil, err
		}
		every, err := parseDuration("every", value("every", "10m"), duration)
		if err != nil || every <= 0 {
			return nil, fmt.Errorf("every must be a positive duration or percentage of --duration")
		}
		width, err := parseDuration("for", value("for", "1m"), duration)
		if err != nil || width <= 0 || width > every {
			return nil, fmt.Errorf("for must be a positive duration or percentage of --duration, no greater than every")
		}
		for elapsed, baseDuration := time.Duration(0), every; elapsed < duration; baseDuration = every - width {
			d := min(baseDuration, duration-elapsed)
			add("base", d, base, base)
			elapsed += d
			d = min(width, duration-elapsed)
			add("spike", d, 1, 1)
			elapsed += d
			if len(p.Phases) > MaxPhases {
				return nil, fmt.Errorf("load exceeds %d phases; increase every or shorten --duration", MaxPhases)
			}
		}
	}
	if _, err := NewPacer(p.Phases); err != nil {
		return nil, err
	}
	return p, nil
}

func parseFraction(key, value string) (float64, error) {
	fraction, err := strconv.ParseFloat(strings.TrimSuffix(value, "%"), 64)
	if strings.HasSuffix(value, "%") {
		fraction /= 100
	}
	if err != nil || math.IsNaN(fraction) || fraction < 0 || fraction > 1 {
		return 0, fmt.Errorf("%s must be a fraction from 0 to 1 or a percentage from 0%% to 100%%", key)
	}
	return fraction, nil
}

func parseDuration(key, value string, total time.Duration) (time.Duration, error) {
	if strings.HasSuffix(value, "%") {
		fraction, err := parseFraction(key, value)
		if err != nil {
			return 0, err
		}
		if fraction == 1 {
			return total, nil
		}
		return time.Duration(float64(total) * fraction), nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration < 0 {
		return 0, fmt.Errorf("%s must be a nonnegative duration (10m) or percentage of --duration (10%%)", key)
	}
	return duration, nil
}
