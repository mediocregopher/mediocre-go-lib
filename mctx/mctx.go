// Package mctx extends the builtin context package to add easy-to-use
// annotation functionality, which is useful for logging and errors.
//
// All functions and methods in this package are thread-safe unless otherwise
// noted.
//
// Annotations
//
// Annotations are a special case of key/values, where the data being
// stored is specifically runtime metadata which would be useful for logging,
// error output, etc... Annotation data might include an IP address of a
// connected client, a userID the client has authenticated as, the primary key
// of a row in a database being queried, etc...
//
package mctx
