package mhttp

import (
	"net/http/httptest"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestAddXForwardedFor(t *T) {
	assertXFF := func(prev []string, ipStr, expected string) massert.Assertion {
		r := httptest.NewRequest("GET", "/", nil)
		for i := range prev {
			r.Header.Add("X-Forwarded-For", prev[i])
		}
		AddXForwardedFor(r, ipStr)
		var a massert.Assertion
		if expected == "" {
			a = massert.Len(r.Header["X-Forwarded-For"], 0)
		} else {
			a = massert.All(
				massert.Len(r.Header["X-Forwarded-For"], 1),
				massert.Equal(expected, r.Header["X-Forwarded-For"][0]),
			)
		}
		return massert.Comment(a, "prev:%#v ipStr:%q", prev, ipStr)
	}

	massert.Fatal(t, massert.All(
		assertXFF(nil, "invalid", ""),
		assertXFF(nil, "::1", ""),
		assertXFF([]string{"8.0.0.0"}, "invalid", "8.0.0.0"),
		assertXFF([]string{"8.0.0.0"}, "::1", "8.0.0.0"),

		assertXFF(nil, "8.0.0.0", "8.0.0.0"),
		assertXFF([]string{"8.0.0.0"}, "8.0.0.1", "8.0.0.0, 8.0.0.1"),
		assertXFF([]string{"8.0.0.0, 8.0.0.1"}, "8.0.0.2", "8.0.0.0, 8.0.0.1, 8.0.0.2"),
		assertXFF([]string{"8.0.0.0, 8.0.0.1", "8.0.0.2"}, "8.0.0.3",
			"8.0.0.0, 8.0.0.1, 8.0.0.2, 8.0.0.3"),
	))
}
