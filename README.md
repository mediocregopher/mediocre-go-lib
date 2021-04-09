# mediocre-go-lib

This is a collection of packages which I use across many of my personal
projects.

## Styleguide

Here are general guidelines I use when making decisions about how code in this
repo should be written. Most of the guidelines I have come up with myself have
to do with package design, since packages are the only thing which have any
rigidity and therefore need any rigid rules.

Everything here are guidelines, not actual rules.

* `gofmt -s`

* https://golang.org/doc/effective_go.html

* https://github.com/golang/go/wiki/CodeReviewComments

* Package names may be abbreviations of a concept, but types, functions, and
  methods should all be full words.

* When deciding if a package should initialize a struct as a value or pointer, a
  good rule is: If it's used as an immutable value it should be a value,
  otherwise it's a pointer. Even if the immutable value implementation has
  internal caching, locks, etc..., that can be hidden as pointers inside the
  struct, but the struct itself can remain a value type.

* A function which takes in a `context.Context` and returns a modified copy of
  that same `context.Context` should have a name prefixed with `With`, e.g.
  `WithValue` or `WithLogger`. Exceptions like `Annotate` do exist.
