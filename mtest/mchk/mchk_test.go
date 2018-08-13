package mchk

import (
	"errors"
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mrand"
)

func TestCheckerRun(t *T) {
	c := Checker{
		Init: func() State { return 0 },
		Next: func(State) Action {
			if mrand.Intn(3) == 0 {
				return Action{Params: -1}
			}
			return Action{Params: 1}
		},
		Apply: func(s State, a Action) (State, error) {
			si := s.(int) + a.Params.(int)
			if si > 5 {
				return nil, errors.New("went over 5")
			}
			return si, nil
		},
		MaxLength: 4,
	}

	// 4 Actions should never be able to go over 5
	if err := c.RunFor(time.Second); err != nil {
		t.Fatal(err)
	}

	// 20 should always go over 5 eventually
	c.MaxLength = 20
	err := c.RunFor(time.Second)
	if err == nil {
		t.Fatal("expected error when maxDepth is 20")
	} else if len(err.(RunErr).Params) < 6 {
		t.Fatalf("strange RunErr when maxDepth is 20: %s", err)
	}

	t.Logf("got expected error with large maxDepth:\n%s", err)
	caseErr := c.RunCase(err.(RunErr).Params...)
	if caseErr == nil || err.Error() != caseErr.Error() {
		t.Fatalf("unexpected caseErr: %v", caseErr)
	}
}
