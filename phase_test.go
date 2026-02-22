package rubric

import "testing"

func TestNewPhase(t *testing.T) {
	tests := map[string]struct {
		id          string
		description string
		weight      float64
		opts        []PhaseOpt
		wantErr     string
	}{
		"valid": {
			id: "p1", description: "Phase", weight: 1.0,
			opts: validPhaseOpts(),
		},
		"large weight": {
			id: "p", description: "Phase", weight: 999.99,
			opts: validPhaseOpts(),
		},
		"empty id": {
			id: "", description: "Phase", weight: 1.0,
			opts:    validPhaseOpts(),
			wantErr: "phase id must not be empty",
		},
		"empty description": {
			id: "p", description: "", weight: 1.0,
			opts:    validPhaseOpts(),
			wantErr: "phase description must not be empty",
		},
		"zero weight": {
			id: "p", description: "Phase", weight: 0,
			opts:    validPhaseOpts(),
			wantErr: "phase weight must be > 0",
		},
		"negative weight": {
			id: "p", description: "Phase", weight: -1,
			opts:    validPhaseOpts(),
			wantErr: "phase weight must be > 0",
		},
		"no signals": {
			id: "p", description: "Phase", weight: 1.0,
			opts:    nil,
			wantErr: "phase must have at least one signal",
		},
		"option error propagated": {
			id: "p", description: "Phase", weight: 1.0,
			opts: []PhaseOpt{func(ph *Phase) error {
				return errSentinel
			}},
			wantErr: "sentinel",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ph, err := NewPhase(tc.id, tc.description, tc.weight, tc.opts...)

			if tc.wantErr != "" {
				mustErr(t, err, tc.wantErr)
				if ph != nil {
					t.Fatal("expected nil phase on error")
				}
				return
			}

			mustNotErr(t, err)
			if ph.ID() != tc.id {
				t.Errorf("ID() = %q, want %q", ph.ID(), tc.id)
			}
			if ph.Description() != tc.description {
				t.Errorf("Description() = %q, want %q", ph.Description(), tc.description)
			}
			if ph.Weight() != tc.weight {
				t.Errorf("Weight() = %v, want %v", ph.Weight(), tc.weight)
			}
		})
	}
}

func TestPhase_GetSignal(t *testing.T) {
	ph := MustNewPhase("p", "Phase", 1.0,
		BuildSignal("a", "Alpha", 0, BuildOutcome("x", "X", 1)),
		BuildSignal("b", "Beta", 0, BuildOutcome("x", "X", 1)),
	)

	t.Run("exists", func(t *testing.T) {
		sig, ok := ph.GetSignal("a")
		if !ok || sig == nil {
			t.Fatal("expected to find signal 'a'")
		}
		if sig.ID() != "a" {
			t.Errorf("ID() = %q, want %q", sig.ID(), "a")
		}
	})

	t.Run("not found", func(t *testing.T) {
		sig, ok := ph.GetSignal("missing")
		if ok || sig != nil {
			t.Fatal("expected not found for 'missing'")
		}
	})
}

func TestMustNewPhase_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustNewPhase("", "bad", 1.0) // empty id
}

func TestMustNewPhase_Success(t *testing.T) {
	ph := MustNewPhase("ok", "Valid", 1.0, validPhaseOpts()...)
	if ph.ID() != "ok" {
		t.Fatalf("ID() = %q, want %q", ph.ID(), "ok")
	}
}

func TestWithSignal(t *testing.T) {
	t.Run("nil phase", func(t *testing.T) {
		sig := MustNewSignal("s", "S", 0, BuildOutcome("a", "A", 1))
		err := WithSignal(sig)(nil)
		mustErr(t, err, "phase must not be nil")
	})

	t.Run("nil signal", func(t *testing.T) {
		ph := &Phase{signals: make(map[string]*Signal)}
		err := WithSignal(nil)(ph)
		mustErr(t, err, "signal must not be nil")
	})

	t.Run("duplicate signal", func(t *testing.T) {
		sig := MustNewSignal("dup", "Dup", 0, BuildOutcome("a", "A", 1))
		ph := &Phase{signals: map[string]*Signal{"dup": sig}}
		err := WithSignal(sig)(ph)
		mustErr(t, err, `signal with id "dup" already exists`)
	})
}

func TestPhase_Signals(t *testing.T) {
	ph := MustNewPhase("p", "Phase", 1.0,
		BuildSignal("c", "Charlie", 0, BuildOutcome("x", "X", 1)),
		BuildSignal("a", "Alpha", 0, BuildOutcome("x", "X", 1)),
		BuildSignal("b", "Beta", 0, BuildOutcome("x", "X", 1)),
	)

	signals := ph.Signals()

	if len(signals) != 3 {
		t.Fatalf("len(Signals()) = %d, want 3", len(signals))
	}

	// Must be sorted by ID.
	wantIDs := []string{"a", "b", "c"}
	for i, want := range wantIDs {
		if signals[i].ID() != want {
			t.Errorf("Signals()[%d].ID() = %q, want %q", i, signals[i].ID(), want)
		}
	}
}

func TestBuildSignal_PropagatesError(t *testing.T) {
	// BuildSignal with an invalid signal id (empty) should propagate.
	opt := BuildSignal("", "bad", 0, validSignalOpts()...)
	ph := &Phase{signals: make(map[string]*Signal)}
	err := opt(ph)
	mustErr(t, err, "signal id must not be empty")
}
