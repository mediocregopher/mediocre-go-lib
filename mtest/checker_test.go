package mtest

import (
	"errors"
	. "testing"
	"time"
)

type testCheckerRunIncr int

func (i testCheckerRunIncr) Apply(s State) (State, error) {
	si := s.(int) + int(i)
	if si > 5 {
		return nil, errors.New("went over 5")
	}
	return si, nil
}

func TestCheckerRun(t *T) {
	c := Checker{
		Init: func() State { return 0 },
		Actions: func(State) []Action {
			return []Action{
				{
					Applyer: testCheckerRunIncr(1),
					Weight:  2,
				},
				{
					Applyer: testCheckerRunIncr(-1),
				},
			}
		},
	}

	// 4 Actions should never be able to go over 5
	if err := c.Run(4, time.Second); err != nil {
		t.Fatal(err)
	}

	// 20 should always go over 5 eventually
	err := c.Run(20, time.Second)
	if err == nil {
		t.Fatal("expected error when maxDepth is 20")
	} else if len(err.(CheckerErr).Applied) < 6 {
		t.Fatalf("strange CheckerErr when maxDepth is 20: %s", err)
	}

	t.Logf("got expected error with large maxDepth:\n%s", err)
	caseErr := c.RunCase(err.(CheckerErr).Applied...)
	if caseErr == nil || err.Error() != caseErr.Error() {
		t.Fatalf("unexpected caseErr: %v", caseErr)
	}
}
