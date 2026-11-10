package load

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var parameterEquals = regexp.MustCompile(`\s*=\s*`)

type Pattern struct {
	Name      string
	Peak      float64
	Duration  time.Duration
	Phases    []Phase
	Canonical string
}

func Parse(value string, duration time.Duration) (pattern *Pattern, err error) {
	var parts []string
	for _, segment := range strings.Split(value, ",") {
		fields := strings.Fields(parameterEquals.ReplaceAllString(segment, "="))
		if len(fields) == 0 {
			fields = []string{""}
		}
		parts = append(parts, fields...)
	}
	name := strings.TrimSpace(parts[0])
	defer func() {
		if err != nil {
			err = fmt.Errorf("%w; e.g. %s", err, Example(name))
		}
	}()
	valid, err := patternParameters(name)
	if err != nil {
		return nil, err
	}
	choices := parameterChoices(valid)
	params := make(map[string]string, len(parts)-1)
	for _, part := range parts[1:] {
		key, value, ok := strings.Cut(part, "=")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if !ok || key == "" || value == "" || strings.Contains(value, "=") {
			return nil, fmt.Errorf("--load parameter %q must use key=value; valid parameters: %s", part, choices)
		}
		if key != "peak" && !slices.Contains(valid, key) {
			suggestion := ""
			for _, candidate := range append([]string{"peak"}, valid...) {
				if len(candidate) > 2 && parameterDistance(candidate, key) <= 1 {
					suggestion = fmt.Sprintf(" (did you mean %q?)", candidate)
					break
				}
			}
			return nil, fmt.Errorf("unknown parameter %q for %s%s; valid parameters: %s", key, name, suggestion, choices)
		}
		if _, ok := params[key]; ok {
			return nil, fmt.Errorf("duplicate --load parameter %q; valid parameters: %s", key, choices)
		}
		params[key] = value
	}
	value, ok := params["peak"]
	if !ok {
		return nil, fmt.Errorf("--load requires peak=, for example %s,peak=120/s; valid parameters: %s", name, choices)
	}
	peak, err := ParsePeak(value)
	if err != nil {
		return nil, fmt.Errorf("%w; valid parameters: %s", err, choices)
	}
	delete(params, "peak")
	pattern, err = Compile(name, peak, duration, params)
	if err != nil {
		return nil, fmt.Errorf("%w; valid parameters: %s", err, choices)
	}
	return pattern, nil
}

func ParsePeak(value string) (float64, error) {
	if strings.Contains(value, "%") {
		return 0, fmt.Errorf("peak does not accept percentages; use a fraction such as 0.1")
	}
	peak, err := strconv.ParseFloat(strings.TrimSuffix(value, "/s"), 64)
	if err != nil || !validLevel(peak) || peak == 0 {
		return 0, fmt.Errorf("peak in --load must be a positive finite iterations/s rate <= %d, for example peak=120/s or peak=120", MaxPeak)
	}
	return peak, nil
}

func patternParameters(name string) ([]string, error) {
	switch name {
	case "steady":
		return nil, nil
	case "ramp-hold-ramp":
		return []string{"up", "down"}, nil
	case "sweep":
		return []string{"from", "to", "steps"}, nil
	case "spike-train":
		return []string{"base", "every", "for"}, nil
	default:
		return nil, fmt.Errorf("unknown load pattern %q; valid patterns: steady, ramp-hold-ramp, sweep, spike-train", name)
	}
}

func parameterChoices(params []string) string {
	return strings.Join(append([]string{"peak"}, params...), ", ")
}

func Compile(name string, peak float64, duration time.Duration, params map[string]string) (*Pattern, error) {
	valid, err := patternParameters(name)
	if err != nil {
		return nil, err
	}
	for key := range params {
		if !slices.Contains(valid, key) {
			return nil, fmt.Errorf("unknown parameter %q for %s; valid parameters: %s", key, name, parameterChoices(valid))
		}
	}
	if duration <= 0 {
		return nil, fmt.Errorf("--load requires a positive --duration")
	}
	if !validLevel(peak) || peak == 0 {
		return nil, fmt.Errorf("--load requires a positive finite peak= <= %d iterations/s", MaxPeak)
	}
	canonical := []string{name, "peak=" + strconv.FormatFloat(peak, 'f', -1, 64) + "/s"}
	value := func(key, fallback string) string {
		v, ok := params[key]
		if !ok {
			v = fallback
		}
		normalized := v
		if number, err := strconv.ParseFloat(v, 64); err == nil {
			normalized = strconv.FormatFloat(number, 'f', -1, 64)
		} else if duration, err := time.ParseDuration(v); err == nil {
			normalized = duration.String()
			if strings.HasSuffix(normalized, "m0s") {
				normalized = strings.TrimSuffix(normalized, "0s")
			}
			if strings.HasSuffix(normalized, "h0m") {
				normalized = strings.TrimSuffix(normalized, "0m")
			}
		}
		canonical = append(canonical, key+"="+normalized)
		return v
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
		up, err := parseDuration("up", value("up", "0.1"), duration)
		if err != nil {
			return nil, err
		}
		down, err := parseDuration("down", value("down", "0.1"), duration)
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
		from, err := parseFraction("from", value("from", "0.25"))
		if err != nil {
			return nil, err
		}
		to, err := parseFraction("to", value("to", "1"))
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
		base, err := parseFraction("base", value("base", "0.4"))
		if err != nil {
			return nil, err
		}
		every, err := parseDuration("every", value("every", "10m"), duration)
		if err != nil {
			return nil, err
		}
		if every <= 0 {
			return nil, fmt.Errorf("every must be a positive duration or fraction of --duration")
		}
		width, err := parseDuration("for", value("for", "1m"), duration)
		if err != nil {
			return nil, err
		}
		if width <= 0 || width > every {
			return nil, fmt.Errorf("for must be a positive duration or fraction of --duration, no greater than every")
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
	p.Canonical = strings.Join(canonical, ",")
	return p, nil
}

func parseFraction(key, value string) (float64, error) {
	if strings.Contains(value, "%") {
		return 0, fmt.Errorf("%s does not accept percentages; use a fraction such as 0.1", key)
	}
	fraction, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(fraction) || fraction < 0 || fraction > 1 {
		return 0, fmt.Errorf("%s must be a fraction from 0 to 1", key)
	}
	return fraction, nil
}

func parseDuration(key, value string, total time.Duration) (time.Duration, error) {
	if _, err := strconv.ParseFloat(value, 64); err == nil || strings.Contains(value, "%") {
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
		return 0, fmt.Errorf("%s must be a nonnegative Go duration (10m) or fraction of --duration (0.1)", key)
	}
	return duration, nil
}

func Example(name string) string {
	switch name {
	case "steady":
		return "--load steady peak=120/s"
	case "ramp-hold-ramp":
		return "--load ramp-hold-ramp peak=120/s up=0.2"
	case "sweep":
		return "--load sweep peak=120/s from=0.25 to=1 steps=4"
	default:
		return "--load spike-train peak=120/s every=5m"
	}
}

func parameterDistance(a, b string) int {
	row := make([]int, len(b)+1)
	for i := range row {
		row[i] = i
	}
	for i := range len(a) {
		previous := row[0]
		row[0] = i + 1
		for j := range len(b) {
			cost := 0
			if a[i] != b[j] {
				cost = 1
			}
			previous, row[j+1] = row[j+1], min(row[j+1]+1, row[j]+1, previous+cost)
		}
	}
	return row[len(b)]
}
