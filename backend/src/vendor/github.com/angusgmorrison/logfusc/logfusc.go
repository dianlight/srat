// Package logfusc provides a generic Secret type that obsufcates all string
// representations of its wrapped value, preventing sensitive data from being
// inadvertently written to output.
//
// It is a lightweight approach to redacting secrets and personally identifiable
// information from logs.
package logfusc

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const redactionTmpl = "logfusc.Secret[%T]{REDACTED}"

// Secret wraps a sensitive value, preventing it from being inadvertently
// written to output. This insures against human error leading to runtime data
// leaks. It is not a secrets manager, and has no cryptographic components.
//
// Satisfies [fmt.Stringer], [fmt.GoStringer], [encoding/json.Marshaler] and
// [encoding/json.Unmarshaler].
//
// Secret is NOT thread-safe, but references to the wrapped value should not be
// retained after instantiation, so this shouldn't be a problem.
type Secret[T any] struct {
	value T
}

// NewSecret returns a new [Secret] containing an instance of T. It is
// recommended to pass a value type, not a pointer, since any retained
// references to the wrapped value won't benefit from Secret's protection.
func NewSecret[T any](value T) Secret[T] {
	return Secret[T]{value}
}

// String renders the Secret and its contents in the format `REDACTED T`, where
// T is the type of the obfuscated value.
func (s Secret[_]) String() string {
	return fmt.Sprintf(redactionTmpl, s.value)
}

// GoString satisfies `fmt.GoStringer`, which controls formatting in response to
// the `%#v` directive, preventing the inner value from being printed.
func (s Secret[_]) GoString() string {
	return s.String()
}

// MarshalJSON satisfies [encoding/json.Marshaler], preventing the inner value from
// being inadvertently marshaled to JSON (e.g. as part of a structured log
// entry).
//
// If the wrapped secret that must be marshaled for transport, call
// [Secret.Expose] to unwrap it.
func (s Secret[_]) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('"')
	buf.WriteString(s.String())
	buf.WriteByte('"')
	return buf.Bytes(), nil
}

// UnmarshalError is returned by [Secret.Unmarshal] if the provided JSON cannot be
// unmarshaled into the the type T wrapped by a Secret. It is returned instead
// of the standard [encoding/json] errors to prevent leakage of the secret
// (however malformed).
type UnmarshalError[T any] struct {
	intendedTarget T // intendedTarget is always the zero value of its type
}

const unmarshalErrorTmpl = "failed to unmarshal Secret[%T] due to malformed JSON; details redacted to avoid leaking wrapped %[1]T"

func (e *UnmarshalError[T]) Error() string {
	return fmt.Sprintf(unmarshalErrorTmpl, e.intendedTarget)
}

// UnmarshalJSON satisfies [encoding/json.Unmarshaler], allowing a sensitive value
// to be unmarshaled directly into a [Secret].
//
// If `data` cannot be unmarshaled into type T, an [UnmarshalError] is returned.
func (s *Secret[T]) UnmarshalJSON(data []byte) error {
	// By convention, unmarshaling "null" is a no-op.
	if string(data) == "null" {
		return nil
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return &UnmarshalError[T]{intendedTarget: value}
	}

	*s = NewSecret(value)
	return nil
}

// Expose returns the wrapped secret for use, at which point it is vulnerable to
// leaking to output.
func (s Secret[T]) Expose() T {
	return s.value
}
