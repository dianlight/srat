package mapper

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/thoas/go-funk"
)

func Map(dst any, src any) error {
	if dst == nil || reflect.ValueOf(dst).Kind() != reflect.Ptr {
		return fmt.Errorf("unsupported destination type: %T", dst)
	}
	if src == nil {
		return nil
	}

	if val, ok := src.(MappableTo); ok {
		match, err := val.To(dst)
		if err != nil {
			return err
		}
		if match {
			return nil
		}
	} else if val, ok := dst.(MappableFrom); ok {
		match, err := val.From(src)
		if err != nil {
			return err
		}
		if match {
			return nil
		}
	}
	v := reflect.ValueOf(src)
	switch reflect.Indirect(v).Kind() {
	case reflect.Map:
		return fromMap(dst, src.(map[string]any))
	case reflect.Slice:
		return fromSlice(dst, src.([]any))
	case reflect.Struct:
		return fromInterface(dst, src)
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return fromNative(dst, src)
	default:
		return fmt.Errorf("Unsupported source type: %T for destination %T", src, dst)
	}
}

func redirectValue(value reflect.Value) reflect.Value {
	for {
		if !value.IsValid() || (value.Kind() != reflect.Ptr && value.Kind() != reflect.Interface) {
			return value
		}

		res := value.Elem()

		// Test for a circular type.
		if res.Kind() == reflect.Ptr && value.Kind() == reflect.Ptr && value.Pointer() == res.Pointer() {
			return value
		}

		if !res.IsValid() && value.Kind() == reflect.Ptr {
			return reflect.Zero(value.Type().Elem())
		}

		value = res
	}
}

func fromMap(dst any, src map[string]any) error {
	v := reflect.ValueOf(dst)
	switch kid := reflect.Indirect(v).Kind(); kid {
	case reflect.Slice:
		for _, item := range src {
			var itemT any
			if err := Map(&itemT, item); err != nil {
				return err
			}
			dst = append(dst.([]any), itemT)
		}
		return nil
	case reflect.Map:
		for k, v := range src {
			var itemT any
			if err := Map(&itemT, v); err != nil {
				return err
			} else {
				src[k] = itemT
			}
		}
		return nil
	case reflect.Struct, reflect.Interface:
		keys := funk.Keys(dst)
		for _, key := range keys.([]string) {
			nvt := redirectValue(reflect.ValueOf(dst)).FieldByName(key)
			x := reflect.ValueOf(src[key])
			if !x.IsValid() {
				continue
			}
			if x.CanConvert(nvt.Type()) {
				nvt.Set(x.Convert(nvt.Type()))
			} else {
				itemT := reflect.New(nvt.Type()).Interface()
				err := Map(itemT, src[key])
				if err != nil {
					return err
				}
				nvt.Set(reflect.ValueOf(itemT).Elem())
			}
		}
		return nil
	default:
		return fmt.Errorf("(fromMap) Unsupported destination type %T for source %T", src, dst)
	}
}

func fromSlice(dst any, src []any) error {
	v := reflect.Indirect(reflect.ValueOf(dst))
	switch v.Kind() {
	case reflect.Slice:
		for _, item := range src {
			itemT := reflect.New(v.Type().Elem()).Interface()
			if err := Map(&itemT, item); err != nil {
				return err
			}
			v = reflect.Append(v, reflect.Indirect(reflect.ValueOf(itemT)))
		}
		reflect.ValueOf(dst).Elem().Set(v)
		return nil
		/*
			case *map[string]any:
				for k, v := range src {
					var itemT any
					if err := Map(&itemT, v); err != nil {
						return err
					} else {
						src[k] = itemT
					}
				}
				return nil
		*/
	default:
		return fmt.Errorf("(fromSlice) Unsupported item type %T for destination %T", src, dst)
	}
}

func fromInterface[T any](dst T, src any) error {
	v := reflect.Indirect(reflect.ValueOf(dst))
	//v := redirectValue(reflect.ValueOf(dst))
	switch v.Kind() {
	case reflect.Struct:
		keys := funk.Keys(dst)
		for _, key := range keys.([]string) {
			nvt := v.FieldByName(key)
			x := reflect.ValueOf(src).FieldByName(key)
			if !x.IsValid() {
				continue
			}
			if x.CanConvert(nvt.Type()) {
				nvt.Set(x.Convert(nvt.Type()))
			} else {
				itemT := reflect.New(nvt.Type()).Interface()
				err := Map(itemT, src)
				if err != nil {
					return err
				}
				nvt.Set(reflect.ValueOf(itemT).Elem())
			}
		}
		return nil
	default:
		return fmt.Errorf("(fromInterface) Unsupported source type: %T for destination %T", src, dst)
	}
}

func fromNative[T any](dst T, src any) error {
	v := reflect.Indirect(reflect.ValueOf(dst))
	//v := redirectValue(reflect.ValueOf(dst))
	switch v.Kind() {
	case reflect.Float64, reflect.Float32:
		val, err := strconv.ParseFloat(fmt.Sprintf("%v", src), v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(val)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(fmt.Sprintf("%v", src), v.Type().Bits(), 64)
		if err != nil {
			return err
		}
		v.SetInt(val)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(fmt.Sprintf("%v", src), v.Type().Bits(), 64)
		if err != nil {
			return err
		}
		v.SetUint(val)
		return nil
	case reflect.String:
		v.SetString(fmt.Sprintf("%v", src))
		return nil
	case reflect.Bool:
		val, err := strconv.ParseBool(fmt.Sprintf("%v", src))
		if err != nil {
			return err
		}
		v.SetBool(val)
		return nil
	default:
		return fmt.Errorf("(fromNative) Unsupported source type: %T for destination %T", src, dst)
	}
}
