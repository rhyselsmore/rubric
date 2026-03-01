package rubric

import (
	"math"
	"sync"
	"testing"
)

func TestNewCandidates(t *testing.T) {
	ca := NewCandidates[string]()
	if ca == nil {
		t.Fatal("NewCandidates returned nil")
	}
}

func TestCandidates_Add(t *testing.T) {
	md := testModel(t)

	t.Run("valid", func(t *testing.T) {
		ca := NewCandidates[string]()
		mustNotErr(t, ca.Add("a", md.Evaluate()))
	})

	t.Run("nil evaluation", func(t *testing.T) {
		ca := NewCandidates[string]()
		err := ca.Add("a", nil)
		mustErr(t, err, "evaluation must not be nil")
	})

	t.Run("duplicate kind", func(t *testing.T) {
		ca := NewCandidates[string]()
		mustNotErr(t, ca.Add("a", md.Evaluate()))
		err := ca.Add("a", md.Evaluate())
		mustErr(t, err, "candidate already exists")
	})
}

func TestCandidates_Add_IntKey(t *testing.T) {
	md := MustNewModel("m", "Model", validModelOpts()...)
	ca := NewCandidates[int]()
	mustNotErr(t, ca.Add(1, md.Evaluate()))
	mustNotErr(t, ca.Add(2, md.Evaluate()))
	err := ca.Add(1, md.Evaluate())
	mustErr(t, err, "candidate already exists")
}

func TestClassify(t *testing.T) {
	t.Run("nil candidates", func(t *testing.T) {
		_, err := Classify[string](nil)
		mustErr(t, err, "candidates must not be nil")
	})

	t.Run("empty candidates", func(t *testing.T) {
		ca := NewCandidates[string]()
		_, err := Classify(ca)
		mustErr(t, err, "at least one candidate must be provided")
	})
}

func TestClassify_SingleCandidate(t *testing.T) {
	md := testModel(t)
	ev := md.Evaluate()
	mustNotErr(t, ev.Set("rhetoric", "othering", "explicit"))

	ca := NewCandidates[string]()
	mustNotErr(t, ca.Add("only", ev))

	cl, err := Classify(ca)
	mustNotErr(t, err)

	winner, score := cl.Best()
	if winner != "only" {
		t.Errorf("Best() winner = %q, want %q", winner, "only")
	}
	if score.Model().ID() != "pipeline" {
		t.Errorf("Best() score model = %q, want %q", score.Model().ID(), "pipeline")
	}
}

func TestClassify_PicksHighest(t *testing.T) {
	md := testModel(t)

	// Content with strong pipeline signals.
	evHigh := md.Evaluate()
	mustNotErr(t, evHigh.Set("rhetoric", "othering", "explicit"))
	mustNotErr(t, evHigh.Set("framing", "delegitimization", "blanket"))
	mustNotErr(t, evHigh.Set("framing", "gateway", "subtle"))

	// Content with weak pipeline signals.
	evLow := md.Evaluate()
	mustNotErr(t, evLow.Set("rhetoric", "othering", "absent"))
	mustNotErr(t, evLow.Set("framing", "gateway", "none"))

	ca := NewCandidates[string]()
	mustNotErr(t, ca.Add("high", evHigh))
	mustNotErr(t, ca.Add("low", evLow))

	cl, err := Classify(ca)
	mustNotErr(t, err)

	winner, bestScore := cl.Best()
	if winner != "high" {
		t.Errorf("winner = %q, want %q", winner, "high")
	}

	lowScore, ok := cl.Score("low")
	if !ok {
		t.Fatal("expected to find score for 'low'")
	}
	if bestScore.Normalized() <= lowScore.Normalized() {
		t.Errorf("high normalized (%v) should be > low normalized (%v)",
			bestScore.Normalized(), lowScore.Normalized())
	}
}

func TestClassify_DifferentModels(t *testing.T) {
	// Two candidates evaluated against different models.
	md1 := MustNewModel("m1", "Model 1",
		BuildPhase("p", "Phase", 1.0,
			BuildSignal("s", "Signal", 0,
				BuildOutcome("a", "A", 100),
			),
		),
	)
	md2 := MustNewModel("m2", "Model 2",
		BuildPhase("p", "Phase", 1.0,
			BuildSignal("s", "Signal", 0,
				BuildOutcome("a", "A", 50),
			),
		),
	)

	ev1 := md1.Evaluate()
	mustNotErr(t, ev1.Set("p", "s", "a"))

	ev2 := md2.Evaluate()
	mustNotErr(t, ev2.Set("p", "s", "a"))

	ca := NewCandidates[string]()
	mustNotErr(t, ca.Add("first", ev1))
	mustNotErr(t, ca.Add("second", ev2))

	cl, err := Classify(ca)
	mustNotErr(t, err)

	// Both should have normalized=1 since each is at its own max.
	for _, kind := range []string{"first", "second"} {
		s, ok := cl.Score(kind)
		if !ok {
			t.Fatalf("score not found for %q", kind)
		}
		if math.Abs(s.Normalized()-1.0) > floatTol {
			t.Errorf("%q Normalized() = %v, want 1.0", kind, s.Normalized())
		}
	}

	// Winner should be "first" since its raw score is higher.
	winner, _ := cl.Best()
	if winner != "first" {
		t.Errorf("winner = %q, want %q", winner, "first")
	}
}

func TestClassification_Scores(t *testing.T) {
	md := testModel(t)

	ca := NewCandidates[string]()
	mustNotErr(t, ca.Add("a", md.Evaluate()))
	mustNotErr(t, ca.Add("b", md.Evaluate()))

	cl, err := Classify(ca)
	mustNotErr(t, err)

	scores := cl.Scores()
	if len(scores) != 2 {
		t.Fatalf("len(Scores()) = %d, want 2", len(scores))
	}

	for _, kind := range []string{"a", "b"} {
		if _, ok := scores[kind]; !ok {
			t.Errorf("Scores() missing key %q", kind)
		}
	}
}

func TestClassification_Score_NotFound(t *testing.T) {
	md := testModel(t)
	ca := NewCandidates[string]()
	mustNotErr(t, ca.Add("a", md.Evaluate()))

	cl, err := Classify(ca)
	mustNotErr(t, err)

	_, ok := cl.Score("missing")
	if ok {
		t.Error("Score('missing') should return false")
	}
}

func TestClassification_String(t *testing.T) {
	md := testModel(t)

	evHigh := md.Evaluate()
	mustNotErr(t, evHigh.Set("rhetoric", "othering", "explicit"))

	evLow := md.Evaluate()

	ca := NewCandidates[string]()
	mustNotErr(t, ca.Add("high", evHigh))
	mustNotErr(t, ca.Add("low", evLow))

	cl, err := Classify(ca)
	mustNotErr(t, err)

	str := cl.String()

	for _, want := range []string{
		"Classification: high",
		">",    // winner marker
		"high", // candidate names
		"low",
		"model: pipeline",
		"evaluated",
		"default",
	} {
		if !containsSubstr(str, want) {
			t.Errorf("String() missing %q:\n%s", want, str)
		}
	}
}

func TestCandidates_ConcurrentAdd(t *testing.T) {
	md := testModel(t)
	ca := NewCandidates[int]()

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ca.Add(i, md.Evaluate())
		}()
	}
	wg.Wait()

	// All 100 should have been added (no duplicates since i is unique).
	cl, err := Classify(ca)
	mustNotErr(t, err)

	if len(cl.Scores()) != 100 {
		t.Errorf("len(Scores()) = %d, want 100", len(cl.Scores()))
	}
}
