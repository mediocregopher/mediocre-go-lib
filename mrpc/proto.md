# Protocol

The mediocre-rpc protocol is an RPC protocol with support for streaming
arbitrary numbers of both request and response objects, byte blobs of unknown
length, and managing request/response debug data.

The protocol itself is carried via the jstream protocol, which is specified and
implemented in this repo.

## General

Common rules and terminology which apply across all subsequent documentation for
this spec:

* An "RPC call", or just "call", is composed of two events: a "request" and a
  "response".

* The entity which initiates the call by sending a request is the "client".

* The entity which serves the call by responding to a request is the "server".

* In all JSON object specs, a field which is not required can be omitted
  entirely, and its value is assumed to be the expected type's empty value (e.g.
  `""` for strings, `0` for numbers, `{}` for objects).

## Debug

Many components of this RPC protocol carry a `debug` field, whose value may be
some arbitrary set of data as desired by the user. The use and purpose of the
`debug` field will be different for everyone, but some example use-cases would
be a unique ID useful for tracing, metadata useful for logging in case of an
error, and request/response timings from both the client and server sides
(useful for determining RTT).

When determining if some piece of data should be considered debug data or part
of the request/response proper, imagine that some piece of code was completely
removing the `debug` field in random places at random times. Your application
should run _identically_ in that scenario as in real-life.

In other words: if some field in `debug` effects the behavior of a call directly
then it should not be carried via `debug`. This could mean duplicating data
between `debug` and the request/response proper, e.g. the IP address of the
client.

## Call request

A call request is defined as being two jstream elements read off the pipe by the
server. Once both elements have been read the request is considered to be
completely consumed and the pipe may be used for a new request.

The two elements of the request stream are specified as follows:

* The first element, the head, is a JSON value with an object containing a
  `name` field, which identifies the call being made, and optionally a `debug`
  field.

* The second element is the argument to the call. This may be a JSON value, a
  byte blob, or an embedded stream containing even more elements, depending on
  the call. It's up to the client and server to coordinate beforehand what to
  expect here.

## Call response

A call response is defined as being two jstream elements read off the pipe by
the client. Once both elements have been read the response is considered to be
completely consumed and the pipe may be used for a new request.

The two elements of the response stream are specified as follows:

* The first element is the response from the call. This may be a JSON value, a
  byte blob, or an embedded stream containing even more elements, depending on
  the call. It's up to the client and server to coordinate beforehand what to
  expect here.

* The second element, the tail, is a JSON value with an object optionally
  containing an `err` field, and optionally containing a `debug` field. The
  value of `err` may be any JSON value which is meaningful to the client and
  server. This element is required even if there's no `err` or `debug` data.

## Pipelining

The protocol allows for the server to begin sending back a response, and even to
send back a complete response, _as soon as_ it receives the request head. In
effect this means that the server can be sending back response data while the
client is still sending request data.

Once the server has sent the response tail it can assume the call has completed
successfully, and the server will ignore all subsequent request data (though it
must still fully read the request body off the pipe in order to use the pipe
again for a new call).

From the client's perspective once the the response tail has been received it
can cancel whatever request body data it's in the process of sending.
