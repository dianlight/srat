package mapper

import (
	"fmt"

	// log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

func Map[T any](dst *T, src any) error {
	switch s := src.(type) {
	case Mappable[T]:
		return s.To(dst)
	case map[string]any:
		return fromMap(dst, s)
	case []any:
		return fromSlice(dst, s)
	default:
		return fromInterface(dst, s)
	}
}

func fromMap[T any](dst *T, src map[string]any) error {
	var rerr error = nil
	defer func() {
		rerr = recover().(error)
	}()
	switch d := (interface{}(&dst)).(type) {
	case *[]T:
		for _, item := range src {
			var itemT T
			if err := Map(&itemT, item); err != nil {
				return err
			}
			*d = append(*d, itemT)
		}
	case *map[string]interface{}:
		funk.ForEach(src, func(k string, v any) {
			var itemT T
			if err := Map(&itemT, v); err != nil {
				panic(err)
			} else {
				src[k] = itemT
			}
		})
	default:
		funk.ForEach(src, func(k string, v any) {
			if err := Map(&d, v); err != nil {
				panic(err)
			}
		})
	}
	return rerr
}

func fromSlice[T any](dst *T, src []any) error {
	var rerr error = nil
	defer func() {
		rerr = recover().(error)
	}()
	switch d := (interface{}(&dst)).(type) {
	case *[]T:
		for _, item := range src {
			var itemT T
			if err := Map(&itemT, item); err != nil {
				return err
			}
			*d = append(*d, itemT)
		}
	case *map[string]interface{}:
		panic(fmt.Errorf("Missing support for slices of maps in fromSlice()"))
		/*
			funk.ForEach(src, func(k string, v any) {
				var itemT T
				if err := Map(&itemT, v); err != nil {
					panic(err)
				} else {
					src[k] = itemT
				}
			})
		*/
	default:
		funk.ForEach(src, func(k string, v any) {
			if err := Map(&d, v); err != nil {
				panic(err)
			}
		})
	}
	return rerr
}

func fromInterface[T any](dst *T, src any) error {
	return fmt.Errorf("Unsupported source type: %T", src)
}
