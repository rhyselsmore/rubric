package rubric

import "testing"

func TestNewModel(t *testing.T) {
	tests := map[string]struct {
		id          string
		description string
		opts        []ModelOpt
		wantErr     string
	}{
		"valid": {
			id: "m1", description: "Model",
			opts: validModelOpts(),
		},
		"multiple phases": {
			id: "m2", description: "Model",
			opts: []ModelOpt{
				BuildPhase("a", "Phase A", 1.0, validPhaseOpts()...),
				BuildPhase("b", "Phase B", 2.0, validPhaseOpts()...),
			},
		},
		"empty id": {
			id: "", description: "Model",
			opts:    validModelOpts(),
			wantErr: "model id must not be empty",
		},
		"empty description": {
			id: "m", description: "",
			opts:    validModelOpts(),
			wantErr: "model description must not be empty",
		},
		"no phases": {
			id: "m", description: "Model",
			opts:    nil,
			wantErr: "model must have at least one phase",
		},
		"option error propagated": {
			id: "m", description: "Model",
			opts: []ModelOpt{func(m *Model) error {
				return errSentinel
			}},
			wantErr: "sentinel",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			md, err := NewModel(tc.id, tc.description, tc.opts...)

			if tc.wantErr != "" {
				mustErr(t, err, tc.wantErr)
				if md != nil {
					t.Fatal("expected nil model on error")
				}
				return
			}

			mustNotErr(t, err)
			if md.ID() != tc.id {
				t.Errorf("ID() = %q, want %q", md.ID(), tc.id)
			}
			if md.Description() != tc.description {
				t.Errorf("Description() = %q, want %q", md.Description(), tc.description)
			}
		})
	}
}

func TestModel_GetPhase(t *testing.T) {
	md := MustNewModel("m", "Model",
		BuildPhase("a", "Alpha", 1.0, validPhaseOpts()...),
		BuildPhase("b", "Beta", 2.0, validPhaseOpts()...),
	)

	t.Run("exists", func(t *testing.T) {
		ph, ok := md.GetPhase("a")
		if !ok || ph == nil {
			t.Fatal("expected to find phase 'a'")
		}
		if ph.ID() != "a" {
			t.Errorf("ID() = %q, want %q", ph.ID(), "a")
		}
	})

	t.Run("not found", func(t *testing.T) {
		ph, ok := md.GetPhase("missing")
		if ok || ph != nil {
			t.Fatal("expected not found for 'missing'")
		}
	})
}

func TestModel_Evaluate(t *testing.T) {
	md := MustNewModel("m", "Model", validModelOpts()...)
	ev := md.Evaluate()
	if ev == nil {
		t.Fatal("Evaluate() returned nil")
	}
}

func TestMustNewModel_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustNewModel("", "bad") // empty id
}

func TestMustNewModel_Success(t *testing.T) {
	md := MustNewModel("ok", "Valid", validModelOpts()...)
	if md.ID() != "ok" {
		t.Fatalf("ID() = %q, want %q", md.ID(), "ok")
	}
}

func TestWithPhase(t *testing.T) {
	t.Run("nil model", func(t *testing.T) {
		ph := MustNewPhase("p", "Phase", 1.0, validPhaseOpts()...)
		err := WithPhase(ph)(nil)
		mustErr(t, err, "model must not be nil")
	})

	t.Run("nil phase", func(t *testing.T) {
		md := &Model{phases: make(map[string]*Phase)}
		err := WithPhase(nil)(md)
		mustErr(t, err, "phase must not be nil")
	})

	t.Run("duplicate phase", func(t *testing.T) {
		ph := MustNewPhase("dup", "Dup", 1.0, validPhaseOpts()...)
		md := &Model{phases: map[string]*Phase{"dup": ph}}
		err := WithPhase(ph)(md)
		mustErr(t, err, `phase with id "dup" already exists`)
	})
}

func TestModel_Phases(t *testing.T) {
	md := MustNewModel("m", "Model",
		BuildPhase("c", "Charlie", 3.0, validPhaseOpts()...),
		BuildPhase("a", "Alpha", 1.0, validPhaseOpts()...),
		BuildPhase("b", "Beta", 2.0, validPhaseOpts()...),
	)

	phases := md.Phases()

	if len(phases) != 3 {
		t.Fatalf("len(Phases()) = %d, want 3", len(phases))
	}

	// Must be sorted by ID.
	wantIDs := []string{"a", "b", "c"}
	for i, want := range wantIDs {
		if phases[i].ID() != want {
			t.Errorf("Phases()[%d].ID() = %q, want %q", i, phases[i].ID(), want)
		}
	}
}

func TestBuildPhase_PropagatesError(t *testing.T) {
	opt := BuildPhase("", "bad", 1.0, validPhaseOpts()...)
	md := &Model{phases: make(map[string]*Phase)}
	err := opt(md)
	mustErr(t, err, "phase id must not be empty")
}
