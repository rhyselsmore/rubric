package rubric

import (
	"math"
	"testing"
)

const floatTol = 1e-9

func TestScore_AllDefaults(t *testing.T) {
	md := testModel(t)
	ev := md.Evaluate()
	s := ev.Score()

	// rhetoric: othering default=10, phase weight=1.5  → 1.5*10 = 15
	// framing: delegitimization default=0, gateway default=0, phase weight=0.8 → 0.8*0 = 0
	// raw = 15
	wantRaw := 15.0

	if math.Abs(s.Raw()-wantRaw) > floatTol {
		t.Errorf("Raw() = %v, want %v", s.Raw(), wantRaw)
	}

	if s.Model().ID() != "pipeline" {
		t.Errorf("Model().ID() = %q, want %q", s.Model().ID(), "pipeline")
	}

	// Signal min/max weights:
	//   othering:          min=-10, max=30
	//   delegitimization:  default=0, outcomes 50,25,10 → min=0, max=50
	//   gateway:           default=0, outcomes 20,25,-10 → min=-10, max=25
	wantMin := 1.5*(-10) + 0.8*(-10+0)   // -15 + -8 = -23
	wantMax := 1.5*(30) + 0.8*(50.0+25.0) // 45 + 60 = 105

	if math.Abs(s.Min()-wantMin) > floatTol {
		t.Errorf("Min() = %v, want %v", s.Min(), wantMin)
	}
	if math.Abs(s.Max()-wantMax) > floatTol {
		t.Errorf("Max() = %v, want %v", s.Max(), wantMax)
	}

	// Normalized = (raw - min) / (max - min) = (15 - (-23)) / (105 - (-23)) = 38/128
	wantNorm := (wantRaw - wantMin) / (wantMax - wantMin)
	if math.Abs(s.Normalized()-wantNorm) > floatTol {
		t.Errorf("Normalized() = %v, want %v", s.Normalized(), wantNorm)
	}

	if s.Normalized() < 0 || s.Normalized() > 1 {
		t.Errorf("Normalized() = %v, expected in [0, 1]", s.Normalized())
	}
}

func TestScore_WithOutcomes(t *testing.T) {
	md := testModel(t)

	tests := map[string]struct {
		outcomes map[string]map[string]string // phase → signal → outcome
		wantRaw  float64
	}{
		"all signals at highest weight": {
			outcomes: map[string]map[string]string{
				"rhetoric": {"othering": "explicit"},
				"framing":  {"delegitimization": "blanket", "gateway": "subtle"},
			},
			wantRaw: 1.5*30 + 0.8*(50+25), // 45 + 60 = 105
		},
		"all signals at lowest weight": {
			outcomes: map[string]map[string]string{
				"rhetoric": {"othering": "absent"},
				"framing":  {"delegitimization": "credible", "gateway": "none"},
			},
			wantRaw: 1.5*(-10) + 0.8*(10+(-10)), // -15 + 0 = -15
		},
		"partial: only rhetoric set": {
			outcomes: map[string]map[string]string{
				"rhetoric": {"othering": "explicit"},
			},
			wantRaw: 1.5*30 + 0.8*(0+0), // 45 + 0 = 45
		},
		"partial: only one framing signal set": {
			outcomes: map[string]map[string]string{
				"framing": {"delegitimization": "selective"},
			},
			// rhetoric defaults: 1.5*10 = 15
			// framing: 0.8*(25 + 0) = 20
			wantRaw: 15 + 20,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ev := md.Evaluate()
			for phase, signals := range tc.outcomes {
				for signal, outcome := range signals {
					mustNotErr(t, ev.Set(phase, signal, outcome))
				}
			}

			s := ev.Score()
			if math.Abs(s.Raw()-tc.wantRaw) > floatTol {
				t.Errorf("Raw() = %v, want %v", s.Raw(), tc.wantRaw)
			}
		})
	}
}

func TestScore_NormalizedBounds(t *testing.T) {
	md := testModel(t)

	t.Run("low score stays in range", func(t *testing.T) {
		ev := md.Evaluate()
		ev.Set("rhetoric", "othering", "absent")
		ev.Set("framing", "delegitimization", "credible")
		ev.Set("framing", "gateway", "none")

		s := ev.Score()
		if s.Normalized() < -floatTol || s.Normalized() > 1+floatTol {
			t.Errorf("Normalized() = %v, out of [0, 1]", s.Normalized())
		}
	})

	t.Run("maximum possible gives 1", func(t *testing.T) {
		ev := md.Evaluate()
		ev.Set("rhetoric", "othering", "explicit")
		ev.Set("framing", "delegitimization", "blanket")
		ev.Set("framing", "gateway", "subtle")

		s := ev.Score()
		if math.Abs(s.Normalized()-1.0) > floatTol {
			t.Errorf("Normalized() = %v, want 1.0", s.Normalized())
		}
	})
}

func TestScore_NormalizedConstantWeights(t *testing.T) {
	// When min == max (all outcomes have the same weight), normalized should be 1.
	md := MustNewModel("flat", "Flat",
		BuildPhase("p", "Phase", 1.0,
			BuildSignal("s", "Signal", 5,
				BuildOutcome("a", "A", 5),
			),
		),
	)
	ev := md.Evaluate()
	s := ev.Score()

	if s.Normalized() != 1.0 {
		t.Errorf("Normalized() = %v, want 1.0 when min == max", s.Normalized())
	}
}

func TestScore_Details(t *testing.T) {
	md := testModel(t)
	ev := md.Evaluate()
	mustNotErr(t, ev.Set("rhetoric", "othering", "explicit"))

	s := ev.Score()
	details := s.Details()

	// testModel has 3 signals total across 2 phases.
	if len(details) != 3 {
		t.Fatalf("len(Details()) = %d, want 3", len(details))
	}

	// Details should be sorted by phase ID then signal ID.
	// Phases: "framing" < "rhetoric"
	// Framing signals: "delegitimization" < "gateway"
	// Rhetoric signals: "othering"
	expectedOrder := []struct{ phase, signal string }{
		{"framing", "delegitimization"},
		{"framing", "gateway"},
		{"rhetoric", "othering"},
	}

	for i, want := range expectedOrder {
		d := details[i]
		if d.Phase().ID() != want.phase || d.Signal().ID() != want.signal {
			t.Errorf("detail[%d] = (%q, %q), want (%q, %q)",
				i, d.Phase().ID(), d.Signal().ID(), want.phase, want.signal)
		}
	}

	// The "othering" signal should have outcome "explicit", others should be default.
	for _, d := range details {
		if d.Signal().ID() == "othering" {
			if d.Outcome().ID() != "explicit" {
				t.Errorf("othering outcome = %q, want %q", d.Outcome().ID(), "explicit")
			}
			if d.Weight() != 30 {
				t.Errorf("othering weight = %v, want 30", d.Weight())
			}
		} else {
			if !d.Outcome().IsDefault() {
				t.Errorf("signal %q should have default outcome, got %q",
					d.Signal().ID(), d.Outcome().ID())
			}
		}
	}
}

func TestScore_String(t *testing.T) {
	md := testModel(t)
	ev := md.Evaluate()
	mustNotErr(t, ev.Set("rhetoric", "othering", "explicit"))

	s := ev.Score()
	str := s.String()

	// Smoke test: check it contains key elements.
	for _, want := range []string{
		"Model: pipeline",
		"Score:",
		"Phase: rhetoric",
		"Phase: framing",
		"othering",
		"explicit (Overt dehumanization or scapegoating)",
		"* = default",
	} {
		if !containsSubstr(str, want) {
			t.Errorf("String() missing %q:\n%s", want, str)
		}
	}
}

func TestScore_SinglePhase_SingleSignal(t *testing.T) {
	// Minimal model to verify exact arithmetic.
	md := MustNewModel("min", "Minimal",
		BuildPhase("p", "Phase", 2.0,
			BuildSignal("s", "Signal", 0,
				BuildOutcome("high", "High", 100),
				BuildOutcome("low", "Low", -50),
			),
		),
	)

	tests := map[string]struct {
		outcome  string
		wantRaw  float64
		wantNorm float64
	}{
		"default": {
			outcome:  "",
			wantRaw:  0,
			wantNorm: (0.0 - 2.0*(-50)) / (2.0*100 - 2.0*(-50)), // 100/300
		},
		"high": {
			outcome:  "high",
			wantRaw:  200,
			wantNorm: 1.0,
		},
		"low": {
			outcome:  "low",
			wantRaw:  -100,
			wantNorm: 0.0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ev := md.Evaluate()
			if tc.outcome != "" {
				mustNotErr(t, ev.Set("p", "s", tc.outcome))
			}
			s := ev.Score()

			if math.Abs(s.Raw()-tc.wantRaw) > floatTol {
				t.Errorf("Raw() = %v, want %v", s.Raw(), tc.wantRaw)
			}
			if math.Abs(s.Normalized()-tc.wantNorm) > floatTol {
				t.Errorf("Normalized() = %v, want %v", s.Normalized(), tc.wantNorm)
			}
		})
	}
}
