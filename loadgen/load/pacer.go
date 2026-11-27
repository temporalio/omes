package load

import (
	"fmt"
	"math"
	"slices"
	"time"
)

const MaxPhases = 1024

// MaxPeak leaves at least 100µs per start to bound scheduler overhead from rate typos.
const MaxPeak = 10_000

// Phase holds a constant rate when From == To, otherwise it ramps linearly.
// Rates are iterations/s. Phases occupy half-open elapsed-time intervals.
type Phase struct {
	Name     string
	Duration time.Duration
	From     float64
	To       float64
}

type Pacer struct {
	phases []Phase
}

func NewPacer(phases []Phase) (*Pacer, error) {
	if len(phases) == 0 || len(phases) > MaxPhases {
		return nil, fmt.Errorf("load requires 1..%d phases", MaxPhases)
	}
	var total time.Duration
	for _, phase := range phases {
		if phase.Duration <= 0 || phase.Duration > time.Duration(math.MaxInt64)-total ||
			!validLevel(phase.From) || !validLevel(phase.To) {
			return nil, fmt.Errorf("invalid load phase %q: require positive duration and finite nonnegative rates <= %d/s", phase.Name, MaxPeak)
		}
		total += phase.Duration
	}
	return &Pacer{phases: slices.Clone(phases)}, nil
}

func validLevel(level float64) bool {
	return level >= 0 && level <= MaxPeak && !math.IsNaN(level)
}

func (p *Pacer) Rate(elapsed time.Duration) float64 {
	if elapsed < 0 {
		return 0
	}
	for _, phase := range p.phases {
		if elapsed < phase.Duration {
			return phase.From + (phase.To-phase.From)*float64(elapsed)/float64(phase.Duration)
		}
		elapsed -= phase.Duration
	}
	return 0
}

// Count returns the number of requested slots at or before elapsed.
func (p *Pacer) Count(elapsed time.Duration) int64 {
	if elapsed < 0 {
		return 0
	}
	if period := p.steadyPeriod(); period > 0 {
		return int64(min(elapsed, p.phases[0].Duration) / period)
	}
	work, _ := p.work(elapsed)
	if work >= math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(math.Floor(work))
}

// Next integrates the requested rate to space starts, including through ramps.
// Missed slots are skipped: the next start is strictly after elapsed. Callers
// consume a slot even when backpressure drops its start.
func (p *Pacer) Next(elapsed time.Duration, started int64) (time.Duration, bool) {
	if len(p.phases) == 1 {
		phase := p.phases[0]
		if elapsed >= phase.Duration {
			return 0, true
		}
		if period := p.steadyPeriod(); period > 0 {
			next := max(started, int64(elapsed/period))
			if next >= int64(phase.Duration/period) {
				return phase.Duration - elapsed, false
			}
			return time.Duration(next+1)*period - elapsed, false
		}
	}
	work, end := p.work(elapsed)
	if elapsed >= end {
		return 0, true
	}
	target := max(float64(started)+1, math.Floor(work)+1)
	var start time.Duration
	for _, phase := range p.phases {
		area := (phase.From + phase.To) * phase.Duration.Seconds() / 2
		if target <= area && area > 0 {
			slope := (phase.To - phase.From) / phase.Duration.Seconds()
			seconds := target / phase.From
			if slope != 0 {
				// This form avoids cancellation for shallow ramps and works from zero.
				seconds = 2 * target / (phase.From + math.Sqrt(max(0, phase.From*phase.From+2*slope*target)))
			}
			due := start + min(phase.Duration, time.Duration(math.Ceil(seconds*float64(time.Second))))
			return max(time.Nanosecond, due-elapsed), false
		}
		target -= area
		start += phase.Duration
	}
	// No further start fits, but zero-rate and fractional-work tails still run.
	return end - elapsed, false
}

func (p *Pacer) steadyPeriod() time.Duration {
	if len(p.phases) == 1 {
		phase := p.phases[0]
		if phase.From == phase.To && phase.From > 0 {
			// Keep whole-nanosecond steady intervals exact through the rate conversion.
			interval := math.Round(float64(time.Second) / phase.From)
			if interval < float64(math.MaxInt64) && float64(time.Second)/interval == phase.From {
				return time.Duration(interval)
			}
		}
	}
	return 0
}

func (p *Pacer) work(elapsed time.Duration) (float64, time.Duration) {
	var work float64
	var end time.Duration
	for _, phase := range p.phases {
		local := max(time.Duration(0), min(elapsed-end, phase.Duration))
		seconds := local.Seconds()
		work += phase.From*seconds + (phase.To-phase.From)*seconds*seconds/(2*phase.Duration.Seconds())
		end += phase.Duration
	}
	return work, end
}
