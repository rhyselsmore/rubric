package rubric

import "testing"

func TestNewOutcome(t *testing.T) {
	tests := map[string]struct {
		id          string
		description string
		weight      float64
		wantErr     string
	}{
		"valid": {
			id: "hit", description: "Hit detected", weight: 5,
		},
		"zero weight": {
			id: "zero", description: "Zero weight", weight: 0,
		},
		"negative weight": {
			id: "neg", description: "Negative", weight: -3.5,
		},
		"empty id": {
			id: "", description: "Something", weight: 1,
			wantErr: "outcome id must not be empty",
		},
		"empty description": {
			id: "x", description: "", weight: 1,
			wantErr: "outcome description must not be empty",
		},
		"reserved default id": {
			id: "default", description: "Not allowed", weight: 1,
			wantErr: `outcome id cannot be "default"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			oc, err := NewOutcome(tc.id, tc.description, tc.weight)

			if tc.wantErr != "" {
				mustErr(t, err, tc.wantErr)
				if oc != nil {
					t.Fatal("expected nil outcome on error")
				}
				return
			}

			mustNotErr(t, err)
			if oc.ID() != tc.id {
				t.Errorf("ID() = %q, want %q", oc.ID(), tc.id)
			}
			if oc.Description() != tc.description {
				t.Errorf("Description() = %q, want %q", oc.Description(), tc.description)
			}
			if oc.Weight() != tc.weight {
				t.Errorf("Weight() = %v, want %v", oc.Weight(), tc.weight)
			}
			if oc.IsDefault() {
				t.Error("IsDefault() = true for non-default outcome")
			}
		})
	}
}

func TestNewOutcome_WithOpts(t *testing.T) {
	errOpt := func(oc *Outcome) error {
		return errSentinel
	}
	_, err := NewOutcome("x", "desc", 1, errOpt)
	mustErr(t, err, "sentinel")
}

func TestMustNewOutcome_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from MustNewOutcome with invalid args")
		}
	}()
	MustNewOutcome("", "bad", 1) // empty id → panic
}

func TestMustNewOutcome_Success(t *testing.T) {
	oc := MustNewOutcome("ok", "Valid", 42)
	if oc.ID() != "ok" {
		t.Fatalf("ID() = %q, want %q", oc.ID(), "ok")
	}
}

func TestOutcome_IsDefault(t *testing.T) {
	// Construct a default outcome the way the production code does.
	oc := &Outcome{id: defaultOutcomeName, description: "default", weight: 0}
	if !oc.IsDefault() {
		t.Fatal("IsDefault() = false for outcome with reserved default id")
	}
}

var errSentinel = errorString("sentinel")

type errorString string

func (e errorString) Error() string { return string(e) }
