package rubric

import (
	"errors"
	"fmt"
)

const defaultOutcomeName = "default"

// OutcomeOpt is a functional option applied when constructing an [Outcome].
type OutcomeOpt func(*Outcome) error

// MustNewOutcome is like [NewOutcome] but panics on error.
func MustNewOutcome(id string, description string, weight float64, opts ...OutcomeOpt) *Outcome {
	oc, err := NewOutcome(id, description, weight, opts...)
	if err != nil {
		panic(err)
	}
	return oc
}

// NewOutcome creates an outcome with the given ID, description, and weight.
// The weight determines the score contribution when this outcome is selected
// for a signal. The ID "default" is reserved and cannot be used.
func NewOutcome(id string, description string, weight float64, opts ...OutcomeOpt) (*Outcome, error) {
	if id == "" {
		return nil, errors.New("rubric: outcome id must not be empty")
	}
	if id == defaultOutcomeName {
		return nil, fmt.Errorf("rubric: outcome id cannot be %q", defaultOutcomeName)
	}
	if description == "" {
		return nil, errors.New("rubric: outcome description must not be empty")
	}
	oc := &Outcome{
		id:          id,
		description: description,
		weight:      weight,
	}
	for _, opt := range opts {
		if err := opt(oc); err != nil {
			return nil, err
		}
	}
	return oc, nil
}

// Outcome represents a possible result for a [Signal]. Each outcome carries a
// weight that determines its score contribution when selected. A signal also
// has an implicit default outcome (with the signal's default weight) that is
// used when no outcome is explicitly reported.
type Outcome struct {
	id          string
	description string
	weight      float64
}

// ID returns the outcome's unique identifier.
func (oc *Outcome) ID() string { return oc.id }

// Description returns a human-readable summary of the outcome.
func (oc *Outcome) Description() string { return oc.description }

// Weight returns the score contribution of this outcome.
func (oc *Outcome) Weight() float64 { return oc.weight }

// IsDefault reports whether this is the implicit default outcome generated
// when no outcome is explicitly reported for a signal.
func (oc *Outcome) IsDefault() bool { return oc.id == defaultOutcomeName }
