# mediocre-go-lib

This is a collection of packages which I use across many of my personal
projects. All packages intended to be used start with an `m`, packages not
starting with `m` are for internal use within this set of packages.

Other third-party packages which integrate into these:

* [merry](github.com/ansel1/merry): used by `mlog` to embed KV logging
  information into `error` instances, it should be assumed that all errors
  returned from these packages are `merry.Error` instances. In cases where a
  package has a specific error it might return and which might be checked for a
  function to perform that equality check will be supplied as part of the
  package.
