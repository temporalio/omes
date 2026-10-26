// Package phaseplan parses and schedules the shared phase timeline used by
// phased activity producers and workers.
package phaseplan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Plan describes one synchronized load and worker-capacity experiment.
type Plan struct {
	// Phases run sequentially in the order supplied by the user.
	Phases []Phase
}

// Phase describes the producer rate and worker profile for one time interval.
type Phase struct {
	// Name is a human-readable label included in logs and errors.
	Name string
	// Duration is the phase's scheduled wall-clock duration.
	Duration time.Duration
	// AddRate is the aggregate number of activities requested per second.
	AddRate float64
	// WorkerProfile is nil when no worker should run. A non-nil value names the
	// OMES worker profile that should run for this phase.
	WorkerProfile *string
}

type rawPlan struct {
	Phases []rawPhase `json:"phases"`
}

type rawPhase struct {
	Name          string          `json:"name"`
	Duration      string          `json:"duration"`
	AddRate       *float64        `json:"addRate"`
	WorkerProfile json.RawMessage `json:"workerProfile"`
}

// Parse validates a JSON experiment plan. Unknown fields are rejected so a
// misspelled control field cannot silently change an experiment.
func Parse(value string) (*Plan, error) {
	var raw rawPlan
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("phase plan must contain exactly one JSON object")
	}

	if len(raw.Phases) == 0 {
		return nil, errors.New("at least one phase is required")
	}

	plan := &Plan{Phases: make([]Phase, 0, len(raw.Phases))}
	hasPositiveRate := false
	for i, rawPhase := range raw.Phases {
		name := strings.TrimSpace(rawPhase.Name)
		if name == "" {
			return nil, fmt.Errorf("phase %d has an empty name", i)
		}
		duration, err := time.ParseDuration(rawPhase.Duration)
		if err != nil {
			return nil, fmt.Errorf("phase %q has invalid duration %q: %w", name, rawPhase.Duration, err)
		}
		if duration <= 0 {
			return nil, fmt.Errorf("phase %q duration must be greater than 0", name)
		}
		if rawPhase.AddRate == nil {
			return nil, fmt.Errorf("phase %q must define addRate", name)
		}
		if *rawPhase.AddRate < 0 {
			return nil, fmt.Errorf("phase %q addRate must be non-negative, got %v", name, *rawPhase.AddRate)
		}
		workerProfile, err := parseWorkerProfile(rawPhase.WorkerProfile)
		if err != nil {
			return nil, fmt.Errorf("phase %q: %w", name, err)
		}
		hasPositiveRate = hasPositiveRate || *rawPhase.AddRate > 0
		plan.Phases = append(plan.Phases, Phase{
			Name:          name,
			Duration:      duration,
			AddRate:       *rawPhase.AddRate,
			WorkerProfile: workerProfile,
		})
	}
	if !hasPositiveRate {
		return nil, errors.New("at least one phase must have an addRate greater than 0")
	}
	return plan, nil
}

func parseWorkerProfile(value json.RawMessage) (*string, error) {
	if len(value) == 0 {
		return nil, errors.New("must define workerProfile; use null for no worker")
	}
	if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return nil, nil
	}
	var profile string
	if err := json.Unmarshal(value, &profile); err != nil {
		return nil, errors.New("workerProfile must be a string or null")
	}
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return nil, errors.New("workerProfile must be non-empty; use null for no worker")
	}
	return &profile, nil
}

// TotalDuration returns the duration of the complete phase timeline.
func (p *Plan) TotalDuration() time.Duration {
	var total time.Duration
	for _, phase := range p.Phases {
		total += phase.Duration
	}
	return total
}

// PhaseStart returns the absolute scheduled start of phase index for an
// experiment beginning at startAt.
func (p *Plan) PhaseStart(startAt time.Time, index int) time.Time {
	start := startAt
	for i := 0; i < index; i++ {
		start = start.Add(p.Phases[i].Duration)
	}
	return start
}

// PhaseEnd returns the absolute scheduled end of phase index for an experiment
// beginning at startAt.
func (p *Plan) PhaseEnd(startAt time.Time, index int) time.Time {
	return p.PhaseStart(startAt, index).Add(p.Phases[index].Duration)
}

// WaitForStart waits for the shared experiment start and rejects a participant
// that finished initialization too late to join the synchronized timeline.
func WaitForStart(ctx context.Context, scheduled time.Time) error {
	if !time.Now().Before(scheduled) {
		return fmt.Errorf("experiment start time %s has already passed", scheduled.UTC().Format(time.RFC3339Nano))
	}
	return WaitUntil(ctx, scheduled)
}

// WaitUntil blocks until the absolute scheduled time. It returns immediately
// when the caller is late so later phases never shift forward.
func WaitUntil(ctx context.Context, scheduled time.Time) error {
	delay := time.Until(scheduled)
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
