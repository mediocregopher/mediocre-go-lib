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

func TestEqual(t *T) {
	Fatal(t, All(
		Equal(1, 1),
		Equal("foo", "foo"),
	))

	Fatal(t, None(
		Equal(1, 2),
		Equal(1, int64(1)),
		Equal(1, uint64(1)),
		Equal("foo", "bar"),
	))
}

func TestNil(t *T) {
	Fatal(t, All(
		Nil(nil),
		Nil([]byte(nil)),
		Nil(map[int]int(nil)),
		Nil((*struct{})(nil)),
		Nil(interface{}(nil)),
		Nil(error(nil)),
	))

	Fatal(t, None(
		Nil(1),
		Nil([]byte("foo")),
		Nil(map[int]int{1: 1}),
		Nil(&struct{}{}),
		Nil(interface{}("hi")),
		Nil(errors.New("some error")),
	))
}

func TestSubset(t *T) {
	Fatal(t, All(
		Subset([]int{1, 2, 3}, []int{}),
		Subset([]int{1, 2, 3}, []int{1}),
		Subset([]int{1, 2, 3}, []int{2}),
		Subset([]int{1, 2, 3}, []int{1, 2}),
		Subset([]int{1, 2, 3}, []int{2, 1}),
		Subset([]int{1, 2, 3}, []int{1, 2, 3}),
		Subset([]int{1, 2, 3}, []int{1, 3, 2}),

		Subset(map[int]int{1: 1, 2: 2}, map[int]int{}),
		Subset(map[int]int{1: 1, 2: 2}, map[int]int{1: 1}),
		Subset(map[int]int{1: 1, 2: 2}, map[int]int{1: 1, 2: 2}),
	))

	Fatal(t, None(
		Subset([]int{}, []int{1, 2, 3}),
		Subset([]int{1, 2, 3}, []int{4}),
		Subset([]int{1, 2, 3}, []int{1, 3, 2, 4}),

		Subset(map[int]int{1: 1, 2: 2}, map[int]int{1: 2}),
		Subset(map[int]int{1: 1, 2: 2}, map[int]int{1: 1, 3: 3}),
	))

	// make sure changes don't retroactively fail the assertion
	m := map[int]int{1: 1, 2: 2}
	a := Subset(m, map[int]int{1: 1})
	m[1] = 2
	Fatal(t, a)
}

func TestHas(t *T) {
	Fatal(t, All(
		Has([]int{1}, 1),
		Has([]int{1, 2}, 1),
		Has([]int{2, 1}, 1),
		Has(map[int]int{1: 1}, 1),
		Has(map[int]int{1: 2}, 2),
		Has(map[int]int{1: 2, 2: 1}, 1),
		Has(map[int]int{1: 2, 2: 2}, 2),
	))

	Fatal(t, None(
		Has([]int{}, 1),
		Has([]int{1}, 2),
		Has([]int{2, 1}, 3),
		Has(map[int]int{}, 1),
		Has(map[int]int{1: 1}, 2),
		Has(map[int]int{1: 2}, 1),
		Has(map[int]int{1: 2, 2: 1}, 3),
	))

	// make sure changes don't retroactively fail the assertion
	m := map[int]int{1: 1}
	a := Has(m, 1)
	m[1] = 2
	Fatal(t, a)
}

func TestHasKey(t *T) {
	Fatal(t, All(
		HasKey(map[int]int{1: 1}, 1),
		HasKey(map[int]int{1: 1, 2: 2}, 1),
		HasKey(map[int]int{1: 1, 2: 2}, 2),
	))

	Fatal(t, None(
		HasKey(map[int]int{}, 1),
		HasKey(map[int]int{2: 2}, 1),
	))

	// make sure changes don't retroactively fail the assertion
	m := map[int]int{1: 1}
	a := HasKey(m, 1)
	delete(m, 1)
	Fatal(t, a)

}

func TestLen(t *T) {
	Fatal(t, All(
		Len([]int(nil), 0),
		Len([]int{}, 0),
		Len([]int{1}, 1),
		Len([]int{1, 2}, 2),
		Len(map[int]int(nil), 0),
		Len(map[int]int{}, 0),
		Len(map[int]int{1: 1}, 1),
		Len(map[int]int{1: 1, 2: 2}, 2),
	))

	Fatal(t, None(
		Len([]int(nil), 1),
		Len([]int{}, 1),
		Len([]int{1}, 0),
		Len([]int{1}, 2),
		Len([]int{1, 2}, 1),
		Len([]int{1, 2}, 3),
		Len(map[int]int(nil), 1),
		Len(map[int]int{}, 1),
	))

	// make sure changes don't retroactively fail the assertion
	m := map[int]int{1: 1}
	a := Len(m, 1)
	m[2] = 2
	Fatal(t, a)
}
