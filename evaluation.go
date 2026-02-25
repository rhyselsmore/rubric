package rubric

import (
	"errors"
	"fmt"
	"sync"
)

// Outcomes maps phase IDs to signal IDs to the selected outcome ID.
type Outcomes map[string]map[string]string

// MustNewEvaluation is like [NewEvaluation] but panics on error.
func MustNewEvaluation(md *Model) *Evaluation {
	eval, err := NewEvaluation(md)
	if err != nil {
		panic(err)
	}
	return eval
}

// NewEvaluation creates a new evaluation for the given [Model]. The returned
// Evaluation starts with no recorded outcomes; use [Evaluation.Set] to record
// observed signal outcomes before calling [Evaluation.Score].
func NewEvaluation(md *Model) (*Evaluation, error) {
	if md == nil {
		return nil, errors.New("rubric: model must not be nil")
	}
	return &Evaluation{
		model:    md,
		outcomes: make(Outcomes),
	}, nil
}

// Evaluation records observed outcomes for a [Model]'s signals. It is safe for
// concurrent use. Signals that are not explicitly set fall back to their
// default outcome when scored.
type Evaluation struct {
	model    *Model
	mu       sync.RWMutex
	outcomes Outcomes // phase ID → signal ID → outcome ID
}

// Set records the observed outcome for a signal within a phase. All three IDs
// (phase, signal, outcome) must refer to entities that exist in the model;
// otherwise an error is returned. Calling Set more than once for the same
// phase/signal pair overwrites the previous outcome.
func (ev *Evaluation) Set(phase string, signal string, outcome string) error {
	ev.mu.Lock()
	defer ev.mu.Unlock()

	// Get Phase
	ph, ok := ev.model.GetPhase(phase)
	if !ok {
		return fmt.Errorf("rubric: phase with id %q not found", phase)
	}

	// Get Signal
	sig, ok := ph.GetSignal(signal)
	if !ok {
		return fmt.Errorf("rubric: signal with id %q not found (phase=%q)", signal, phase)
	}

	// Get Outcome
	oc, ok := sig.GetOutcome(outcome)
	if !ok {
		return fmt.Errorf("rubric: outcome with id %q not found (phase=%q, signal=%q)", outcome, phase, signal)
	}

	// Ensure Phase
	if _, exists := ev.outcomes[phase]; !exists {
		ev.outcomes[phase] = make(map[string]string)
	}
	ev.outcomes[phase][signal] = oc.id

	return nil
}

// getOutcomes returns a deep copy of the recorded outcomes, safe to read
// without holding the lock.
func (ev *Evaluation) getOutcomes() Outcomes {
	ev.mu.RLock()
	defer ev.mu.RUnlock()

	out := make(map[string]map[string]string, len(ev.outcomes))
	for phase, signals := range ev.outcomes {
		out[phase] = make(map[string]string)
		for signal, outcome := range signals {
			out[phase][signal] = outcome
		}
	}
	return out
}
