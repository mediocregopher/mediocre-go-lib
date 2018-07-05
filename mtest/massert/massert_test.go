package massert

import . "testing"

func TestAssertions(t *T) {
	a := Equal(1, 1)
	b := Equal(2, 2)
	if err := (Assertions{a, b}).Assert(); err != nil {
		t.Fatalf("first Assertions shouldn't return error, returned: %s", err)
	}

	c := Comment(Equal(3, 3), "this part would succeed")
	c = Comment(Not(c), "but it's being wrapped in a not, so it then won't")

	aa := New()
	aa.Add(a)
	aa.Add(b)
	aa.Add(c)
	err := aa.Assert()
	if err == nil {
		t.Fatalf("second Assertions should have returned an error, returned nil")
	}
	t.Logf("got expected second Assertions error:\n%s", err)
}
