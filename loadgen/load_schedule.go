package loadgen

import (
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen/load"
	"go.uber.org/zap"
)

type loadSchedule struct {
	pacer      *load.Pacer
	phases     []load.Phase
	logger     *zap.SugaredLogger
	phaseIndex int
	phaseEnd   time.Duration
	consumed   int64
	dropped    int64
}

func newLoadSchedule(config RunConfiguration, logger *zap.SugaredLogger) (*loadSchedule, error) {
	pattern := config.LoadPattern
	if pattern == nil {
		return nil, nil
	}
	pacer, err := load.NewPacer(pattern.Phases)
	if err != nil {
		return nil, err
	}
	var phases []string
	var start time.Duration
	for _, phase := range pattern.Phases {
		phases = append(phases, fmt.Sprintf("%s [%s,%s) %g→%g iterations/s", phase.Name, start, start+phase.Duration, phase.From, phase.To))
		start += phase.Duration
	}
	logger.Infow("Load pattern", "pattern", pattern.Name, "peak", pattern.Peak, "duration", config.Duration, "phases", phases)
	return &loadSchedule{pacer: pacer, phases: pattern.Phases, logger: logger, phaseIndex: -1}, nil
}

func (s *loadSchedule) next(elapsed time.Duration) (wait time.Duration, boundary, done bool) {
	if s.advance(elapsed) {
		return 0, false, true
	}
	s.drop(s.pacer.Count(elapsed))
	wait, done = s.pacer.Next(elapsed, s.consumed)
	boundary = s.phaseEnd-elapsed < wait
	return min(wait, s.phaseEnd-elapsed), boundary, done
}

func (s *loadSchedule) take(elapsed time.Duration) bool {
	if s.advance(elapsed) {
		return false
	}
	count := s.pacer.Count(elapsed)
	if count <= s.consumed {
		return false
	}
	if s.pacer.Rate(elapsed) == 0 {
		s.drop(count)
		return false
	}
	s.drop(count - 1)
	s.consumed++
	return true
}

func (s *loadSchedule) advance(elapsed time.Duration) (done bool) {
	if s.phaseIndex == len(s.phases) {
		return true
	}
	for elapsed >= s.phaseEnd {
		if s.phaseIndex >= 0 {
			s.drop(s.pacer.Count(s.phaseEnd - time.Nanosecond))
		}
		s.reportDrops()
		s.phaseIndex++
		if s.phaseIndex == len(s.phases) {
			return true
		}
		start := s.phaseEnd
		s.phaseEnd += s.phases[s.phaseIndex].Duration
		s.logger.Infow("Load phase", "phase", s.phases[s.phaseIndex].Name, "phaseIndex", s.phaseIndex+1,
			"requestedIterationsPerSecond", s.pacer.Rate(start))
	}
	return false
}

func (s *loadSchedule) drop(count int64) {
	if count > s.consumed {
		s.dropped += count - s.consumed
		s.consumed = count
	}
}

func (s *loadSchedule) reportDrops() {
	if s.dropped > 0 {
		s.logger.Warnw("Load phase dropped starts; raise --max-concurrent or lower --peak",
			"phase", s.phases[s.phaseIndex].Name, "phaseIndex", s.phaseIndex+1, "dropped", s.dropped)
		s.dropped = 0
	}
}
