// Package rubric provides a weighted scoring framework for multi-phase signal
// evaluation and candidate classification.
//
// A [Model] defines a scoring rubric as a hierarchy: Model → Phase → Signal → Outcome.
// Each [Phase] groups related signals and carries a weight that scales its
// contribution to the overall score. Each [Signal] represents an observable
// indicator with a default weight (used when no outcome is reported) and one or
// more named [Outcome] values, each with its own weight.
//
// To evaluate a model, create an [Evaluation] with [NewEvaluation], record
// observed outcomes with [Evaluation.Set], then call [Evaluation.Score] to
// compute raw and normalized scores.
//
// To compare multiple candidates against different models, use [Candidates] and
// [Classify].
package rubric

import (
	"errors"
	"fmt"
	"sort"
)

// ModelOpt is a functional option applied when constructing a [Model].
type ModelOpt func(*Model) error

// BuildPhase constructs a new [Phase] with the given parameters and adds it to
// the [Model] being built. It is a convenience that combines [NewPhase] and
// [WithPhase] into a single [ModelOpt].
func BuildPhase(id string, description string, weight float64, opts ...PhaseOpt) ModelOpt {
	return func(m *Model) error {
		ph, err := NewPhase(id, description, weight, opts...)
		if err != nil {
			return err
		}
		return WithPhase(ph)(m)
	}
}

// WithPhase returns a [ModelOpt] that adds a pre-constructed [Phase] to the
// [Model]. It returns an error if a phase with the same ID already exists.
func WithPhase(ph *Phase) ModelOpt {
	return func(m *Model) error {
		if m == nil {
			return errors.New("rubric: model must not be nil")
		}
		if ph == nil {
			return errors.New("rubric: phase must not be nil")
		}
		if _, exists := m.phases[ph.id]; exists {
			return fmt.Errorf("rubric: phase with id %q already exists", ph.id)
		}
		m.phases[ph.id] = ph
		return nil
	}
}

// MustNewModel is like [NewModel] but panics on error.
func MustNewModel(id string, description string, opts ...ModelOpt) *Model {
	md, err := NewModel(id, description, opts...)
	if err != nil {
		panic(err)
	}
	return md
}

// NewModel creates a scoring [Model] with the given ID and description. At
// least one [Phase] must be added via the provided options; otherwise an error
// is returned. IDs and descriptions must be non-empty.
func NewModel(id string, description string, opts ...ModelOpt) (*Model, error) {
	if id == "" {
		return nil, errors.New("rubric: model id must not be empty")
	}
	if description == "" {
		return nil, errors.New("rubric: model description must not be empty")
	}
	md := &Model{
		id:          id,
		description: description,
		phases:      make(map[string]*Phase),
	}
	for _, opt := range opts {
		if err := opt(md); err != nil {
			return nil, err
		}
	}
	if len(md.phases) == 0 {
		return nil, errors.New("rubric: model must have at least one phase")
	}
	return md, nil
}

// Model is the top-level scoring rubric. It contains one or more weighted
// phases, each of which contains weighted signals. Use [NewModel] or
// [MustNewModel] to construct one.
type Model struct {
	id          string
	description string
	phases      map[string]*Phase
}

// ID returns the model's unique identifier.
func (md *Model) ID() string { return md.id }

// Description returns a human-readable summary of the model.
func (md *Model) Description() string { return md.description }

// Evaluate creates a new [Evaluation] for this model. It panics if the model
// is nil (which cannot happen on a properly constructed Model).
func (md *Model) Evaluate() *Evaluation {
	return MustNewEvaluation(md)
}

// GetPhase returns the phase with the given ID and true, or nil and false if
// no such phase exists.
func (md *Model) GetPhase(id string) (*Phase, bool) {
	ph, ok := md.phases[id]
	return ph, ok
}

// Phases returns all phases in the model, sorted by ID.
func (md *Model) Phases() []*Phase {
	out := make([]*Phase, 0, len(md.phases))
	for _, ph := range md.phases {
		out = append(out, ph)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}
