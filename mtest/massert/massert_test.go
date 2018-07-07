package massert

import (
	"errors"
	. "testing"
)

func succeed() Assertion {
	return newAssertion(func() error { return nil }, "Succeed", 0)
}

func fail() Assertion {
	return newAssertion(func() error { return errors.New("failure") }, "Fail", 0)
}

func TestNot(t *T) {
	if err := Not(succeed()).Assert(); err == nil {
		t.Fatal("Not(succeed()) should have failed")
	}

	if err := Not(fail()).Assert(); err != nil {
		t.Fatal(err)
	}
}

func TestAny(t *T) {
	if err := Any().Assert(); err == nil {
		t.Fatal("empty Any should fail")
	}

	if err := Any(succeed(), succeed()).Assert(); err != nil {
		t.Fatal(err)
	}

	if err := Any(succeed(), fail()).Assert(); err != nil {
		t.Fatal(err)
	}

	if err := Any(fail(), fail()).Assert(); err == nil {
		t.Fatal("Any should have failed with all inner fail Assertions")
	}
}

func TestAnyOne(t *T) {
	if err := AnyOne().Assert(); err == nil {
		t.Fatal("empty AnyOne should fail")
	}

	if err := AnyOne(succeed(), succeed()).Assert(); err == nil {
		t.Fatal("AnyOne with two succeeds should fail")
	}

	if err := AnyOne(succeed(), fail()).Assert(); err != nil {
		t.Fatal(err)
	}

	if err := AnyOne(fail(), fail()).Assert(); err == nil {
		t.Fatal("AnyOne should have failed with all inner fail Assertions")
	}
}

func TestAll(t *T) {
	if err := All().Assert(); err != nil {
		t.Fatal(err)
	}

	if err := All(succeed(), succeed()).Assert(); err != nil {
		t.Fatal(err)
	}

	if err := All(succeed(), fail()).Assert(); err == nil {
		t.Fatal("All should have failed with one inner fail Assertion")
	}

	if err := All(fail(), fail()).Assert(); err == nil {
		t.Fatal("All should have failed with all inner fail Assertions")
	}
}

func TestNone(t *T) {
	if err := None().Assert(); err != nil {
		t.Fatal(err)
	}

	if err := None(succeed(), succeed()).Assert(); err == nil {
		t.Fatal("None should have failed with all inner succeed Assertions")
	}

	if err := None(succeed(), fail()).Assert(); err == nil {
		t.Fatal("None should have failed with one inner succeed Assertion")
	}

	if err := None(fail(), fail()).Assert(); err != nil {
		t.Fatal(err)
	}
}

// TODO pointers, structs, slices, maps, nils
func TestEqual(t *T) {
	if err := All(
		Equal(1, 1),
		Equal(1, int64(1)),
		Equal(1, uint64(1)),
		Equal("foo", "foo"),
	).Assert(); err != nil {
		t.Fatal(err)
	}

	if err := None(
		Equal(1, 2),
		Equal(1, int64(2)),
		Equal(1, uint64(2)),
		Equal("foo", "bar"),
	).Assert(); err != nil {
		t.Fatal(err)
	}
}

func TestExactly(t *T) {
	if err := All(
		Exactly(1, 1),
		Exactly("foo", "foo"),
	).Assert(); err != nil {
		t.Fatal(err)
	}

	if err := None(
		Exactly(1, 2),
		Exactly(1, int64(1)),
		Exactly(1, uint64(1)),
		Exactly("foo", "bar"),
	).Assert(); err != nil {
		t.Fatal(err)
	}
}
