package rubric

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// NewCandidates creates an empty [Candidates] collection.
func NewCandidates[T comparable]() *Candidates[T] {
	return &Candidates[T]{
		all: make(map[T]*Evaluation),
	}
}

// Candidates holds a set of evaluations keyed by a comparable label type T.
// It preserves insertion order so that ties in [Classify] are broken by the
// order candidates were added. It is safe for concurrent use.
type Candidates[T comparable] struct {
	mu    sync.RWMutex
	order []T
	all   map[T]*Evaluation
}

// Add registers an evaluation under the given label. It returns an error if
// an evaluation for that label already exists or if ev is nil.
func (ca *Candidates[T]) Add(kind T, ev *Evaluation) error {
	if ev == nil {
		return errors.New("rubric: evaluation must not be nil")
	}

	ca.mu.Lock()
	defer ca.mu.Unlock()

	if _, exists := ca.all[kind]; exists {
		return fmt.Errorf("rubric: candidate already exists for kind %v", kind)
	}
	ca.order = append(ca.order, kind)
	ca.all[kind] = ev
	return nil
}

func (ca *Candidates[T]) getOrdered() []T {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	out := make([]T, len(ca.order))
	copy(out, ca.order)
	return out
}

func (ca *Candidates[T]) get(kind T) (*Evaluation, bool) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	ev, ok := ca.all[kind]
	return ev, ok
}

// Classify scores every candidate in the collection and returns a
// [Classification] identifying the winner (highest normalized score). At
// least one candidate must be present; otherwise an error is returned.
func Classify[T comparable](candidates *Candidates[T]) (Classification[T], error) {
	if candidates == nil {
		return Classification[T]{}, errors.New("rubric: candidates must not be nil")
	}

	order := candidates.getOrdered()
	if len(order) == 0 {
		return Classification[T]{}, errors.New("rubric: at least one candidate must be provided")
	}

	scores := make(map[T]Score, len(order))
	var winner T
	var best Score

	for i, kind := range order {
		ev, _ := candidates.get(kind)
		s := ev.Score()
		scores[kind] = s

		if i == 0 || s.Normalized() > best.Normalized() {
			winner = kind
			best = s
		}
	}

	return Classification[T]{
		winner: winner,
		order:  order,
		scores: scores,
	}, nil
}

// Classification is the immutable result of [Classify]. It holds the scores
// for every candidate and identifies the winner.
type Classification[T comparable] struct {
	winner T
	order  []T
	scores map[T]Score
}

// Best returns the winning candidate label and its [Score].
func (c Classification[T]) Best() (T, Score) { return c.winner, c.scores[c.winner] }

// Scores returns a map of all candidate labels to their scores.
func (c Classification[T]) Scores() map[T]Score { return c.scores }

// Score returns the score for a specific candidate label and true, or the zero
// value and false if no such candidate exists.
func (c Classification[T]) Score(kind T) (Score, bool) {
	s, ok := c.scores[kind]
	return s, ok
}

// String returns a human-readable summary of the classification, ranking all
// candidates by normalized score.
func (c Classification[T]) String() string {
	var b strings.Builder

	bestKind, bestScore := c.Best()
	fmt.Fprintf(&b, "Classification: %v (normalized=%.4f)\n", bestKind, bestScore.Normalized())
	b.WriteString(strings.Repeat("=", 80))
	b.WriteByte('\n')

	// Rank by normalized score descending.
	type ranked struct {
		kind  T
		score Score
	}
	ranks := make([]ranked, 0, len(c.order))
	for _, kind := range c.order {
		ranks = append(ranks, ranked{kind: kind, score: c.scores[kind]})
	}
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].score.Normalized() > ranks[j].score.Normalized()
	})

	for i, r := range ranks {
		marker := " "
		if r.kind == bestKind {
			marker = ">"
		}
		fmt.Fprintf(&b, "\n%s #%d  %-20v  normalized=%.4f  raw=%6.2f  [%.2f, %.2f]\n",
			marker, i+1,
			r.kind,
			r.score.Normalized(),
			r.score.Raw(),
			r.score.Min(),
			r.score.Max(),
		)

		// Show per-model detail summary: count of default vs evaluated signals.
		var evaluated, defaulted int
		for _, d := range r.score.Details() {
			if d.outcome.IsDefault() {
				defaulted++
			} else {
				evaluated++
			}
		}
		fmt.Fprintf(&b, "     model: %s | signals: %d evaluated, %d default\n",
			r.score.Model().ID(), evaluated, defaulted)
	}

	b.WriteString(strings.Repeat("=", 80))
	b.WriteByte('\n')

	return b.String()
}
