package rubric

import "testing"

func TestNewSignal(t *testing.T) {
	tests := map[string]struct {
		id            string
		description   string
		defaultWeight float64
		opts          []SignalOpt
		wantErr       string
	}{
		"valid single outcome": {
			id: "s1", description: "Signal", defaultWeight: 5,
			opts: []SignalOpt{BuildOutcome("a", "A", 10)},
		},
		"valid multiple outcomes": {
			id: "s2", description: "Signal", defaultWeight: 0,
			opts: []SignalOpt{
				BuildOutcome("a", "A", 10),
				BuildOutcome("b", "B", -5),
			},
		},
		"empty id": {
			id: "", description: "Signal", defaultWeight: 0,
			opts:    validSignalOpts(),
			wantErr: "signal id must not be empty",
		},
		"empty description": {
			id: "s", description: "", defaultWeight: 0,
			opts:    validSignalOpts(),
			wantErr: "signal description must not be empty",
		},
		"no outcomes": {
			id: "s", description: "Signal", defaultWeight: 0,
			opts:    nil,
			wantErr: "signal must have at least one outcome",
		},
		"option returns error": {
			id: "s", description: "Signal", defaultWeight: 0,
			opts: []SignalOpt{func(sig *Signal) error {
				return errSentinel
			}},
			wantErr: "sentinel",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			sig, err := NewSignal(tc.id, tc.description, tc.defaultWeight, tc.opts...)

			if tc.wantErr != "" {
				mustErr(t, err, tc.wantErr)
				if sig != nil {
					t.Fatal("expected nil signal on error")
				}
				return
			}

			mustNotErr(t, err)
			if sig.ID() != tc.id {
				t.Errorf("ID() = %q, want %q", sig.ID(), tc.id)
			}
			if sig.Description() != tc.description {
				t.Errorf("Description() = %q, want %q", sig.Description(), tc.description)
			}
		})
	}
}

func TestSignal_MinMaxWeight(t *testing.T) {
	tests := map[string]struct {
		defaultWeight float64
		outcomes      []SignalOpt
		wantMin       float64
		wantMax       float64
	}{
		"default is middle": {
			defaultWeight: 5,
			outcomes: []SignalOpt{
				BuildOutcome("low", "Low", -10),
				BuildOutcome("high", "High", 20),
			},
			wantMin: -10, wantMax: 20,
		},
		"default is lowest": {
			defaultWeight: -100,
			outcomes: []SignalOpt{
				BuildOutcome("a", "A", 0),
				BuildOutcome("b", "B", 50),
			},
			wantMin: -100, wantMax: 50,
		},
		"default is highest": {
			defaultWeight: 100,
			outcomes: []SignalOpt{
				BuildOutcome("a", "A", 0),
				BuildOutcome("b", "B", 50),
			},
			wantMin: 0, wantMax: 100,
		},
		"single outcome equal to default": {
			defaultWeight: 10,
			outcomes: []SignalOpt{
				BuildOutcome("same", "Same", 10),
			},
			wantMin: 10, wantMax: 10,
		},
		"all negative": {
			defaultWeight: -5,
			outcomes: []SignalOpt{
				BuildOutcome("a", "A", -10),
				BuildOutcome("b", "B", -1),
			},
			wantMin: -10, wantMax: -1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			sig, err := NewSignal("s", "Signal", tc.defaultWeight, tc.outcomes...)
			mustNotErr(t, err)

			if sig.MinWeight() != tc.wantMin {
				t.Errorf("MinWeight() = %v, want %v", sig.MinWeight(), tc.wantMin)
			}
			if sig.MaxWeight() != tc.wantMax {
				t.Errorf("MaxWeight() = %v, want %v", sig.MaxWeight(), tc.wantMax)
			}
		})
	}
}

func TestSignal_DefaultOutcome(t *testing.T) {
	sig := MustNewSignal("s", "Signal", 42, BuildOutcome("a", "A", 10))

	def := sig.DefaultOutcome()
	if def == nil {
		t.Fatal("DefaultOutcome() returned nil")
	}
	if !def.IsDefault() {
		t.Error("DefaultOutcome().IsDefault() = false")
	}
	if def.Weight() != 42 {
		t.Errorf("DefaultOutcome().Weight() = %v, want 42", def.Weight())
	}
}

func TestSignal_GetOutcome(t *testing.T) {
	sig := MustNewSignal("s", "Signal", 0,
		BuildOutcome("a", "Alpha", 10),
		BuildOutcome("b", "Beta", 20),
	)

	t.Run("exists", func(t *testing.T) {
		oc, ok := sig.GetOutcome("a")
		if !ok || oc == nil {
			t.Fatal("expected to find outcome 'a'")
		}
		if oc.ID() != "a" {
			t.Errorf("ID() = %q, want %q", oc.ID(), "a")
		}
	})

	t.Run("not found", func(t *testing.T) {
		oc, ok := sig.GetOutcome("missing")
		if ok || oc != nil {
			t.Fatal("expected not found for 'missing'")
		}
	})
}

func TestMustNewSignal_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustNewSignal("", "bad", 0) // empty id
}

func TestMustNewSignal_Success(t *testing.T) {
	sig := MustNewSignal("ok", "Valid", 0, BuildOutcome("a", "A", 1))
	if sig.ID() != "ok" {
		t.Fatalf("ID() = %q, want %q", sig.ID(), "ok")
	}
}

func TestWithOutcome(t *testing.T) {
	t.Run("nil signal", func(t *testing.T) {
		opt := WithOutcome(MustNewOutcome("a", "A", 1))
		err := opt(nil)
		mustErr(t, err, "signal must not be nil")
	})

	t.Run("nil outcome", func(t *testing.T) {
		opt := WithOutcome(nil)
		// Need a real signal to call the opt on; build one manually.
		sig := &Signal{outcomes: make(map[string]*Outcome)}
		err := opt(sig)
		mustErr(t, err, "outcome must not be nil")
	})

	t.Run("duplicate outcome", func(t *testing.T) {
		oc := MustNewOutcome("dup", "Dup", 1)
		sig := &Signal{outcomes: map[string]*Outcome{"dup": oc}}
		err := WithOutcome(oc)(sig)
		mustErr(t, err, `outcome with id "dup" already exists`)
	})
}

func TestSignal_Outcomes(t *testing.T) {
	sig := MustNewSignal("s", "Signal", 0,
		BuildOutcome("c", "Charlie", 30),
		BuildOutcome("a", "Alpha", 10),
		BuildOutcome("b", "Beta", 20),
	)

	outcomes := sig.Outcomes()

	if len(outcomes) != 3 {
		t.Fatalf("len(Outcomes()) = %d, want 3", len(outcomes))
	}

	// Must be sorted by ID and must not include the default outcome.
	wantIDs := []string{"a", "b", "c"}
	for i, want := range wantIDs {
		if outcomes[i].ID() != want {
			t.Errorf("Outcomes()[%d].ID() = %q, want %q", i, outcomes[i].ID(), want)
		}
		if outcomes[i].IsDefault() {
			t.Errorf("Outcomes()[%d] should not be the default outcome", i)
		}
	}
}

func TestBuildOutcome_PropagatesError(t *testing.T) {
	// BuildOutcome with an invalid outcome id (empty) should propagate error.
	opt := BuildOutcome("", "bad", 0)
	sig := &Signal{outcomes: make(map[string]*Outcome)}
	err := opt(sig)
	mustErr(t, err, "outcome id must not be empty")
}
