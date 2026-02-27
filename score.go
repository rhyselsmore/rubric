package rubric

import (
	"fmt"
	"sort"
	"strings"
)

// Detail captures how a single signal contributed to the score.
type Detail struct {
	phase   *Phase
	signal  *Signal
	outcome *Outcome
	weight  float64
}

// Phase returns the phase this detail belongs to.
func (d Detail) Phase() *Phase { return d.phase }

// Signal returns the signal that was scored.
func (d Detail) Signal() *Signal { return d.signal }

// Outcome returns the outcome that was selected (or the default).
func (d Detail) Outcome() *Outcome { return d.outcome }

// Weight returns the effective weight contributed by this signal.
func (d Detail) Weight() float64 { return d.weight }

// Score is the immutable result of scoring an Evaluation.
type Score struct {
	model      *Model
	raw        float64
	normalized float64
	min        float64
	max        float64
	details    []Detail
}

// Model returns the model that was scored.
func (s Score) Model() *Model { return s.model }

// Raw returns the un-normalized weighted sum of all signal contributions.
func (s Score) Raw() float64 { return s.raw }

// Normalized returns the score mapped to [0, 1] using min-max normalization.
// If min equals max (all outcomes have the same weight), Normalized returns 1.
func (s Score) Normalized() float64 { return s.normalized }

// Min returns the theoretical minimum raw score (all signals at their lowest
// weight).
func (s Score) Min() float64 { return s.min }

// Max returns the theoretical maximum raw score (all signals at their highest
// weight).
func (s Score) Max() float64 { return s.max }

// Details returns per-signal scoring breakdowns, ordered by phase then signal.
func (s Score) Details() []Detail { return s.details }

// String returns a human-readable multi-line summary of the score, including
// per-phase and per-signal breakdowns.
func (s Score) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Model: %s (%s)\n", s.model.ID(), s.model.Description())
	fmt.Fprintf(&b, "Score: %.4f (normalized) | %.2f (raw) | range [%.2f, %.2f]\n",
		s.normalized, s.raw, s.min, s.max)
	b.WriteString(strings.Repeat("-", 80))
	b.WriteByte('\n')

	var currentPhase string
	for _, d := range s.details {
		if d.phase.ID() != currentPhase {
			currentPhase = d.phase.ID()
			fmt.Fprintf(&b, "\nPhase: %s (weight=%.2f) - %s\n",
				d.phase.ID(), d.phase.Weight(), d.phase.Description())
		}

		marker := " "
		if d.outcome.IsDefault() {
			marker = "*"
		}

		outcomeLabel := d.outcome.ID()
		if !d.outcome.IsDefault() {
			outcomeLabel = fmt.Sprintf("%s (%s)", d.outcome.ID(), d.outcome.Description())
		}

		fmt.Fprintf(&b, "  %s %-20s -> %-30s  weight=%6.2f  [min=%6.2f, max=%6.2f]\n",
			marker,
			d.signal.ID(),
			outcomeLabel,
			d.weight,
			d.signal.minWeight,
			d.signal.maxWeight,
		)
	}

	b.WriteString(strings.Repeat("-", 80))
	b.WriteByte('\n')
	b.WriteString("* = default (no outcome reported)\n")

	return b.String()
}

// Score takes a snapshot of the evaluation's current outcomes and computes a
// [Score]. Signals without a recorded outcome use their default weight. The
// raw score is the sum of (phase weight × signal outcome weight) across all
// phases, and the normalized score maps it to [0, 1] via min-max
// normalization.
func (ev *Evaluation) Score() Score {
	outcomes := ev.getOutcomes()

	var raw, minP, maxP float64
	var details []Detail

	// Sorted phase iteration for deterministic output.
	phaseIDs := make([]string, 0, len(ev.model.phases))
	for id := range ev.model.phases {
		phaseIDs = append(phaseIDs, id)
	}
	sort.Strings(phaseIDs)

	for _, phaseID := range phaseIDs {
		ph := ev.model.phases[phaseID]

		// Sorted signal iteration for deterministic output.
		signalIDs := make([]string, 0, len(ph.signals))
		for id := range ph.signals {
			signalIDs = append(signalIDs, id)
		}
		sort.Strings(signalIDs)

		var phaseRaw, phaseMin, phaseMax float64

		for _, signalID := range signalIDs {
			sig := ph.signals[signalID]

			oc := sig.defaultOutcome
			weight := oc.weight

			if phaseOutcomes, ok := outcomes[phaseID]; ok {
				if ocID, ok := phaseOutcomes[signalID]; ok {
					if matched, ok := sig.outcomes[ocID]; ok {
						weight = matched.weight
						oc = matched
					}
				}
			}

			phaseRaw += weight
			phaseMin += sig.minWeight
			phaseMax += sig.maxWeight

			details = append(details, Detail{
				phase:   ph,
				signal:  sig,
				outcome: oc,
				weight:  weight,
			})
		}

		raw += ph.weight * phaseRaw
		minP += ph.weight * phaseMin
		maxP += ph.weight * phaseMax
	}

	normalized := 1.0
	if maxP != minP {
		normalized = (raw - minP) / (maxP - minP)
	}

	return Score{
		model:      ev.model,
		raw:        raw,
		normalized: normalized,
		min:        minP,
		max:        maxP,
		details:    details,
	}
}
