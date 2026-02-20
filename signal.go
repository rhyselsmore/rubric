package rubric

import (
	"errors"
	"fmt"
	"sort"
)

// SignalOpt is a functional option applied when constructing a [Signal].
type SignalOpt func(*Signal) error

// BuildOutcome constructs a new [Outcome] with the given parameters and adds
// it to the [Signal] being built. It is a convenience that combines
// [NewOutcome] and [WithOutcome] into a single [SignalOpt].
func BuildOutcome(id string, description string, weight float64, opts ...OutcomeOpt) SignalOpt {
	return func(sig *Signal) error {
		oc, err := NewOutcome(id, description, weight, opts...)
		if err != nil {
			return err
		}
		return WithOutcome(oc)(sig)
	}
}

// WithOutcome returns a [SignalOpt] that adds a pre-constructed [Outcome] to
// the [Signal]. It returns an error if an outcome with the same ID already
// exists.
func WithOutcome(oc *Outcome) SignalOpt {
	return func(sig *Signal) error {
		if sig == nil {
			return errors.New("rubric: signal must not be nil")
		}
		if oc == nil {
			return errors.New("rubric: outcome must not be nil")
		}
		if _, exists := sig.outcomes[oc.id]; exists {
			return fmt.Errorf("rubric: outcome with id %q already exists", oc.id)
		}
		sig.outcomes[oc.id] = oc
		return nil
	}
}

// MustNewSignal is like [NewSignal] but panics on error.
func MustNewSignal(id string, description string, defaultWeight float64, opts ...SignalOpt) *Signal {
	prd, err := NewSignal(id, description, defaultWeight, opts...)
	if err != nil {
		panic(err)
	}
	return prd
}

// NewSignal creates a signal with the given ID, description, and default
// weight. The defaultWeight is the score contribution when no outcome has been
// reported for this signal during evaluation. At least one [Outcome] must be
// added via the provided options; otherwise an error is returned.
//
// After construction, [Signal.MinWeight] and [Signal.MaxWeight] reflect the
// range across the default weight and all outcome weights.
func NewSignal(id string, description string, defaultWeight float64, opts ...SignalOpt) (*Signal, error) {
	if id == "" {
		return nil, errors.New("rubric: signal id must not be empty")
	}
	if description == "" {
		return nil, errors.New("rubric: signal description must not be empty")
	}
	defaultOutcome := &Outcome{
		id:          defaultOutcomeName,
		description: "No outcome reported",
		weight:      defaultWeight,
	}

	sig := &Signal{
		id:             id,
		description:    description,
		defaultOutcome: defaultOutcome,
		minWeight:      defaultWeight,
		maxWeight:      defaultWeight,
		outcomes:       make(map[string]*Outcome),
	}

	// apply options
	for _, opt := range opts {
		if err := opt(sig); err != nil {
			return nil, err
		}
	}

	if len(sig.outcomes) == 0 {
		return nil, errors.New("rubric: signal must have at least one outcome")
	}
	// calculate weights
	for _, outcome := range sig.outcomes {
		if outcome.weight > sig.maxWeight {
			sig.maxWeight = outcome.weight
		}
		if outcome.weight < sig.minWeight {
			sig.minWeight = outcome.weight
		}
	}

	return sig, nil
}

// Signal represents a single observable indicator within a [Phase]. It
// carries a default outcome (used when no outcome is explicitly reported) and
// one or more named outcomes, each with their own weight. The min and max
// weights are computed at construction time from the full set of possible
// outcome weights.
type Signal struct {
	id             string
	description    string
	defaultOutcome *Outcome
	minWeight      float64
	maxWeight      float64
	outcomes       map[string]*Outcome
}

// ID returns the signal's unique identifier.
func (sig *Signal) ID() string { return sig.id }

// Description returns a human-readable summary of the signal.
func (sig *Signal) Description() string { return sig.description }

// DefaultOutcome returns the outcome used when no outcome is explicitly
// reported during evaluation.
func (sig *Signal) DefaultOutcome() *Outcome { return sig.defaultOutcome }

// MinWeight returns the minimum possible weight across all outcomes (including
// the default).
func (sig *Signal) MinWeight() float64 { return sig.minWeight }

// MaxWeight returns the maximum possible weight across all outcomes (including
// the default).
func (sig *Signal) MaxWeight() float64 { return sig.maxWeight }

// GetOutcome returns the outcome with the given ID and true, or nil and false
// if no such outcome exists.
func (sig *Signal) GetOutcome(id string) (*Outcome, bool) {
	oc, ok := sig.outcomes[id]
	return oc, ok
}

// Outcomes returns all explicitly added outcomes for the signal, sorted by ID.
// The default outcome is not included; use [Signal.DefaultOutcome] for that.
func (sig *Signal) Outcomes() []*Outcome {
	out := make([]*Outcome, 0, len(sig.outcomes))
	for _, oc := range sig.outcomes {
		out = append(out, oc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}
