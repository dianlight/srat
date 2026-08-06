package dto

import (
	"encoding/json"
	"reflect"

	"github.com/angusgmorrison/logfusc"
	"github.com/danielgtaylor/huma/v2"
)

type Secret[T any] logfusc.Secret[T]

func NewSecret[T any](value T) Secret[T] {
	return Secret[T](logfusc.NewSecret(value))
}

func (s Secret[T]) Expose() T {
	return logfusc.Secret[T](s).Expose()
}

func (o Secret[T]) Schema(r huma.Registry) *huma.Schema {
	// Get the schema for the underlying type T, not the Secret wrapper
	var zero T
	return r.Schema(reflect.TypeOf(zero), true, "")
}

func (o Secret[T]) MarshalJSON() ([]byte, error) {
	// Secrets are write-only: never serialize their value (or the logfusc
	// Go type literal) in responses. Emit null so consumers see the field
	// as absent without leaking any representation of the secret (issue #904).
	return json.Marshal(nil)
}

func (o *Secret[T]) UnmarshalJSON(data []byte) error {
	return (*logfusc.Secret[T])(o).UnmarshalJSON(data)
}

func (o Secret[T]) GoString() string {
	return logfusc.Secret[T](o).GoString()
}

func (o Secret[T]) String() string {
	return logfusc.Secret[T](o).String()
}

/*
func (o Secret[T]) MarshalText() ([]byte, error) {
	return logfusc.Secret[T](o).MarshalText()
}

func (o *Secret[T]) UnmarshalText(data []byte) error {
	return (*logfusc.Secret[T])(o).UnmarshalText(data)
}
*/
