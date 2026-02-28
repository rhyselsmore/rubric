package rubric

import "testing"

// validSignalOpts returns a minimal set of SignalOpts that satisfy the "at
// least one outcome" requirement.
func validSignalOpts() []SignalOpt {
	return []SignalOpt{
		BuildOutcome("present", "Present", 10),
	}
}

// validPhaseOpts returns a minimal set of PhaseOpts that satisfy the "at least
// one signal" requirement.
func validPhaseOpts() []PhaseOpt {
	return []PhaseOpt{
		BuildSignal("sig", "A signal", 0, validSignalOpts()...),
	}
}

// validModelOpts returns a minimal set of ModelOpts that satisfy the "at least
// one phase" requirement.
func validModelOpts() []ModelOpt {
	return []ModelOpt{
		BuildPhase("ph", "A phase", 1.0, validPhaseOpts()...),
	}
}

// testModel builds the model used by most evaluation and scoring tests. It
// models detection of alt-right pipeline content with two phases and three
// signals that exercise a range of positive, negative, and zero weights.
//
//	rhetoric (weight 1.5) — Rhetorical Techniques
//	  othering       default=10  outcomes: explicit=30, absent=-10
//
//	framing (weight 0.8) — Narrative Framing
//	  delegitimization  default=0   outcomes: blanket=50, selective=25, credible=10
//	  gateway           default=0   outcomes: overt=20, subtle=25, none=-10
func testModel(t *testing.T) *Model {
	t.Helper()
	return MustNewModel("pipeline", "Radicalization Pipeline Scoring",
		BuildPhase("rhetoric", "Rhetorical Techniques", 1.5,
			BuildSignal("othering", "Us-vs-Them Language", 10,
				BuildOutcome("explicit", "Overt dehumanization or scapegoating", 30),
				BuildOutcome("absent", "No othering language detected", -10),
			),
		),
		BuildPhase("framing", "Narrative Framing", 0.8,
			BuildSignal("delegitimization", "Source Delegitimization", 0,
				BuildOutcome("blanket", "Blanket rejection of mainstream sources", 50),
				BuildOutcome("selective", "Selective distrust of specific outlets", 25),
				BuildOutcome("credible", "Cites credible, verifiable sources", 10),
			),
			BuildSignal("gateway", "Gateway Potential", 0,
				BuildOutcome("overt", "Overtly extreme, easily identified", 20),
				BuildOutcome("subtle", "Appears moderate but funnels toward extremism", 25),
				BuildOutcome("none", "No gateway characteristics", -10),
			),
		),
	)
}

// mustNotErr fails the test if err is non-nil.
func mustNotErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// mustErr fails the test if err is nil, and optionally checks that the error
// message contains substr.
func mustErr(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if substr != "" && !containsSubstr(err.Error(), substr) {
		t.Fatalf("error %q does not contain %q", err.Error(), substr)
	}
}

func containsSubstr(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && searchSubstr(s, substr)
}

func searchSubstr(s, substr string) bool {
	for i := range len(s) - len(substr) + 1 {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
