# mediocre-go-lib

This is a collection of packages which I use across many of my personal
projects. All packages intended to be used start with an `m`, packages not
starting with `m` are for internal use within this set of packages.

## Usage notes

* In general, all checking of equality of errors, e.g. `err == io.EOF`, done on
  errors returned from the packages in this project should be done using
  `merr.Equal`, e.g. `merr.Equal(err, io.EOF)`. The `merr` package is used to
  wrap errors and embed further metadata in them, like stack traces and so
  forth.

## Styleguide

Here are general guidelines I use when making decisions about how code in this
repo should be written. Most of the guidelines I have come up with myself have
to do with package design, since packages are the only thing which have any
rigidity and therefore need any rigid rules.

Everything here are guidelines, not actual rules.

* `gofmt -s`

* https://golang.org/doc/effective_go.html

* https://github.com/golang/go/wiki/CodeReviewComments

* When deciding if a package should initialize a struct as a value or pointer, a
  good rule is: If it's used as an immutable value it should be a value,
  otherwise it's a pointer. Even if the immutable value implementation has
  internal caching, locks, etc..., that can be hidden as pointers inside the
  struct, but the struct itself can remain a value type.
