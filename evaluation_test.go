package rubric

import (
	"sync"
	"testing"
)

func TestNewEvaluation(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		md := testModel(t)
		ev, err := NewEvaluation(md)
		mustNotErr(t, err)
		if ev == nil {
			t.Fatal("expected non-nil evaluation")
		}
	})

	t.Run("nil model", func(t *testing.T) {
		ev, err := NewEvaluation(nil)
		mustErr(t, err, "model must not be nil")
		if ev != nil {
			t.Fatal("expected nil evaluation on error")
		}
	})
}

func TestMustNewEvaluation_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustNewEvaluation(nil)
}

func TestMustNewEvaluation_Success(t *testing.T) {
	md := testModel(t)
	ev := MustNewEvaluation(md)
	if ev == nil {
		t.Fatal("expected non-nil evaluation")
	}
}

func TestEvaluation_Set(t *testing.T) {
	md := testModel(t)

	tests := map[string]struct {
		phase   string
		signal  string
		outcome string
		wantErr string
	}{
		"valid": {
			phase: "rhetoric", signal: "othering", outcome: "explicit",
		},
		"unknown phase": {
			phase: "bogus", signal: "othering", outcome: "explicit",
			wantErr: `phase with id "bogus" not found`,
		},
		"unknown signal": {
			phase: "rhetoric", signal: "bogus", outcome: "explicit",
			wantErr: `signal with id "bogus" not found`,
		},
		"unknown outcome": {
			phase: "rhetoric", signal: "othering", outcome: "bogus",
			wantErr: `outcome with id "bogus" not found`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ev := md.Evaluate()
			err := ev.Set(tc.phase, tc.signal, tc.outcome)

			if tc.wantErr != "" {
				mustErr(t, err, tc.wantErr)
				return
			}
			mustNotErr(t, err)
		})
	}
}

func TestEvaluation_Set_OverwritesPrevious(t *testing.T) {
	md := testModel(t)
	ev := md.Evaluate()

	mustNotErr(t, ev.Set("rhetoric", "othering", "explicit"))
	mustNotErr(t, ev.Set("rhetoric", "othering", "absent"))

	s := ev.Score()
	for _, d := range s.Details() {
		if d.Signal().ID() == "othering" {
			if d.Outcome().ID() != "absent" {
				t.Errorf("expected overwritten outcome 'absent', got %q", d.Outcome().ID())
			}
			return
		}
	}
	t.Fatal("othering signal not found in score details")
}

func TestEvaluation_Set_ConcurrentSafety(t *testing.T) {
	md := testModel(t)
	ev := md.Evaluate()

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			ev.Set("rhetoric", "othering", "explicit")
		}()
		go func() {
			defer wg.Done()
			ev.Score()
		}()
	}
	wg.Wait()
}
