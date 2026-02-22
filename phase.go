package rubric

import (
	"errors"
	"fmt"
	"sort"
)

// PhaseOpt is a functional option applied when constructing a [Phase].
type PhaseOpt func(*Phase) error

// BuildSignal constructs a new [Signal] with the given parameters and adds it
// to the [Phase] being built. It is a convenience that combines [NewSignal] and
// [WithSignal] into a single [PhaseOpt].
func BuildSignal(id string, description string, defaultWeight float64, opts ...SignalOpt) PhaseOpt {
	return func(m *Phase) error {
		sig, err := NewSignal(id, description, defaultWeight, opts...)
		if err != nil {
			return err
		}
		return WithSignal(sig)(m)
	}
}

// WithSignal returns a [PhaseOpt] that adds a pre-constructed [Signal] to the
// [Phase]. It returns an error if a signal with the same ID already exists.
func WithSignal(sig *Signal) PhaseOpt {
	return func(ph *Phase) error {
		if ph == nil {
			return errors.New("rubric: phase must not be nil")
		}
		if sig == nil {
			return errors.New("rubric: signal must not be nil")
		}
		if _, exists := ph.signals[sig.id]; exists {
			return fmt.Errorf("rubric: signal with id %q already exists", sig.id)
		}
		ph.signals[sig.id] = sig
		return nil
	}
}

// MustNewPhase is like [NewPhase] but panics on error.
func MustNewPhase(id string, description string, weight float64, opts ...PhaseOpt) *Phase {
	ph, err := NewPhase(id, description, weight, opts...)
	if err != nil {
		panic(err)
	}
	return ph
}

// NewPhase creates a scoring phase with the given ID, description, and weight.
// The weight must be positive and determines how much this phase's signal
// scores are scaled in the overall model score. At least one [Signal] must be
// added via the provided options; otherwise an error is returned.
func NewPhase(id string, description string, weight float64, opts ...PhaseOpt) (*Phase, error) {
	if id == "" {
		return nil, errors.New("rubric: phase id must not be empty")
	}
	if description == "" {
		return nil, errors.New("rubric: phase description must not be empty")
	}
	if weight <= 0 {
		return nil, errors.New("rubric: phase weight must be > 0")
	}
	ph := &Phase{
		id:          id,
		description: description,
		weight:      weight,
		signals:     make(map[string]*Signal),
	}
	for _, opt := range opts {
		if err := opt(ph); err != nil {
			return nil, err
		}
	}
	if len(ph.signals) == 0 {
		return nil, errors.New("rubric: phase must have at least one signal")
	}
	return ph, nil
}

// Phase is a weighted group of signals within a [Model]. During scoring, the
// sum of a phase's signal weights is multiplied by the phase weight.
type Phase struct {
	id          string
	description string
	weight      float64
	signals     map[string]*Signal
}

// ID returns the phase's unique identifier.
func (ph *Phase) ID() string { return ph.id }

// Description returns a human-readable summary of the phase.
func (ph *Phase) Description() string { return ph.description }

// Weight returns the phase's weight, which scales all signal scores within it.
func (ph *Phase) Weight() float64 { return ph.weight }

// GetSignal returns the signal with the given ID and true, or nil and false if
// no such signal exists.
func (ph *Phase) GetSignal(id string) (*Signal, bool) {
	sig, ok := ph.signals[id]
	return sig, ok
}

// Signals returns all signals in the phase, sorted by ID.
func (ph *Phase) Signals() []*Signal {
	out := make([]*Signal, 0, len(ph.signals))
	for _, sig := range ph.signals {
		out = append(out, sig)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}
