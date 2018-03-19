# Protocol

The mediocre-rpc protocol is designed to operate over nearly any network
protocol. It exists entirely in the data layer, and only relies on being carried
by a protocol which supports a request/response paradaigm, and garauntees
order/reception. This includes HTTP and raw TCP sockets, and likely many others.

RPC calls and responses are JSON encoded objects. The protocol supports single
object calls/responses, as well as streaming multiple objects for either. There
is also support for streaming raw byte blobs.

## General

A couple common rules which apply across all subsequent documentation for this
spec:

* In all JSON object specs, a field which is not required can be omitted
  entirely, and its value is assumed to be the expected type's empty value (e.g.
  `""` for strings, `0` for numbers, `{}` for objects, `null` for any-JSON
  types).

* For multiple JSON objects appearing back-to-back on the wire there may or may
  not be white-space separating them.

## Calls

### Single call

A single call looks like the following:

	{
		"method":"methodName (required)",
		"args":"anyJSON",
		"debug":{ "foo":"anyJSON" }
	}

`method` is required and indicates the name of the method being called. Its
value has no restrictions on the protocol level, it's up to the caller and
handler to know ahead of time which methods are available.

`args` are the arguments to the method, and can be anything.

`debug` is metadata about the call which can be made accessible to the handler
for purposes of tracing, logging, etc... `debug` must be a JSON object. A good
rule to know if something should be debug or part of the arguments is that if
any `debug` field were to be deleted on any request the response wouldn't
change.  If the response were to change then that field should be in `args`.

### Stream call

A stream call consists of a leading JSON object which looks like a single call,
a body of zero or more JSON objects containing single elements of a stream, and
a tail which indicates the end of the stream.

A stream method call whose body is all JSON strings might look this (newlines
added for clarity, they are optional in the protocol):

	{
		"method":"methodName (required)",
		"args":"anyJSON",
		"debug":{ "foo":"anyJSON" },
		"streamStart":true,
		"streamLen":3
	}

	{ "el":"anyJSON" }

	{ "elBytesFrame":"RANDFRAME" }
	RANDFRAMEsome very cool arbitrary bytes
	which may contain whitespace or anythingRANDFRAME

	{
		"elBytesFrame":"RANDFRAME",
		"elBytesLen":10
	}
	RANDFRAMEsome-bytesRANDFRAME

	{
		"args":"anyJSON",
		"debug":{ "foo":"anyJSON" },
		"streamEnd":true
	}

The head is the first JSON value in the stream. It looks and operates much like a
single call's head, but with the added `streamStart` field. The `streamLen`
field is optional, but may be used if the number of elements in the stream is
known beforehand.

Each element in the stream is a JSON object, and can be either a single JSON
value or a blob of arbitrary bytes.

If the element is a JSON value the JSON object will have an `el` field whose
value is the JSON value.

If the element is a blob of arbitrary bytes the JSON object will have an
`elBytesFrame` field whose value is a set of random alphanumeric characters
which will be used to prefix and suffix the bytes. Following the JSON object may
be some whitespace, and then the arbitrary bytes with the `elBytesFrame`
prefixed and suffixed immediately around it. The JSON object for the element may
optionally have the `elBytesLen` field if the length of the blob is known
beforehand. The arbitrary bytes _must_ be prefixed/suffixed by the frame even if
`elBytesLen` is given.

TODO elBytesFrame size recommendation

The tail is the last JSON value in the stream and indicates the stream
has ended. It is required even if `streamLen` was given in the head. The tail
can also have its own `args` and `debug`, independent of the head's but subject
to the same rules.

## Response

### Single response

A single response looks like the following:

	{
		"res":"anyJSON",
		"err":{
			"msg":"some presumably helpful string",
			"meta":"anyJSON"
		},
		"debug":{ "foo":"anyJSON" }
	}

`res` is the response from the call, and can be anything.

`err` is mutually exclusive with `res` (if one is set the other should be `null`
or unset). The `msg` is be any arbitrary string. `meta` is optional and contains
any extra information which might be actionable by the client receiving the
error.

`debug` is metadata about the response which can be made accessible to the
caller for purposes of tracing, logging, etc... `debug` must be a JSON object. A
good rule to know if something should be debug or part of the results is that
if any `debug` field were to be deleted on any response the client would act in
the same way. If the client's subsequent actions were to change then that field
should be in `res`.

### Stream response

A stream response looks and acts very similar to a stream call, and the
documentation for the stream call can be referenced for details on the
following:

	{
		"res":"anyJSON",
		"debug":{ "foo":"anyJSON" },
		"streamStart":true,
		"streamLen":3
	}

	{ "el":"anyJSON" }

	{ "elBytesFrame":"RANDFRAME" }
	RANDFRAMEsome very cool arbitrary bytes
	which may contain whitespace or anythingRANDFRAME

	{
		"elBytesFrame":"RANDFRAME",
		"elBytesLen":10
	}
	RANDFRAMEsome-bytesRANDFRAME

	{
		"args":"anyJSON",
		"debug":{ "foo":"anyJSON" },
		"streamEnd":true
	}

The one note specific to stream responses is that a stream response _cannot_
contain an `err` field.

## Network details

In the case of a stream call the handler may send back a response before the
stream has been completely sent. In this case the client should ignore the fact
that the stream wasn't completely sent and return the response as it receives
it.

In the case of a stream call and a stream response both the client and handler
can be sending their respective streams to the other simultaneously. As in the
previous case, if the handler ends the stream with the tail object the caller
should treat the call as successfully completed.

The general rule is: If the client receives either a single response or the tail
of a stream response then the call should be treated as completed.

## TODO

* In the case of pipelining then short-circuiting ought to be implemented for
  the case of the stream response having been completed but the stream call
  hasn't. In that case the stream call should short-circuit the stream so the
  connection can be reused asap

* Figure out the naming

* Len is weird, cause if a stream is short-circuited then the length will have
  just been a hint

* Maybe just make bytes a thing which can be bolted onto any of the JSON
  objects, either the single, head, elements, or tail.

* Maybe instead of merely defining a single-level stream, do something where all
  elements in the stream are the same (either a json value or bytes), and
  additionally each can declare that it is the beginning of a stream of further
  objects. That gets weird with both `method` and `err`, but it's kinda nice in
  that it would potentially simplify the code interface greatly.
