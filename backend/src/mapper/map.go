package mapper

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/thoas/go-funk"
)

func Map(dst any, src any) error {
	if dst == nil || reflect.ValueOf(dst).Kind() != reflect.Ptr {
		return fmt.Errorf("unsupported destination type: %T a pointer is required!", dst)
	}
	if src == nil {
		return nil
	}
	if reflect.ValueOf(src).Kind() == reflect.Ptr {
		src = reflect.ValueOf(src).Elem().Interface()
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
		return fromMap(dst, src /*.(map[string]any)*/)
	case reflect.Slice:
		return fromSlice(dst, src)
	case reflect.Struct:
		return fromStruct(dst, src)
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

func fromMap(dst any, src any /*map[string]any*/) error {
	v := reflect.Indirect(reflect.ValueOf(dst))
	vsrc := reflect.Indirect(reflect.ValueOf(src))
	switch kid := v.Kind(); kid {
	case reflect.Slice:
		for i := 0; i < vsrc.Len(); i++ {
			//for _, item := range src {
			var item = vsrc.Index(i).Interface()
			var itemT any
			if err := Map(&itemT, item); err != nil {
				return err
			}
			dst = append(dst.([]any), itemT)
		}
		return nil
	case reflect.Map:
		isrc := vsrc.MapRange()
		for isrc.Next() {
			ks := isrc.Key()
			vs := isrc.Value()
			if vs.CanConvert(reflect.TypeOf(v).Elem()) {
				v.SetMapIndex(ks, vs.Convert(reflect.TypeOf(v).Elem()))
			} else {
				itemT := reflect.New(v.Elem().Type()).Interface()
				err := Map(itemT, vs.Interface())
				if err != nil {
					return err
				}
				v.SetMapIndex(ks, reflect.ValueOf(itemT).Elem())

			}
		}
		return nil
	case reflect.Struct, reflect.Interface:
		for _, key := range vsrc.MapKeys() {
			nvt := redirectValue(v).FieldByName(key.Interface().(string))
			x := vsrc.MapIndex(key)
			if !nvt.IsValid() {
				continue
			}
			if x.CanConvert(nvt.Type()) {
				nvt.Set(x.Convert(nvt.Type()))
			} else {
				itemT := reflect.New(nvt.Type()).Interface()
				err := Map(itemT, vsrc.MapIndex(key).Interface())
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

func fromSlice(dst any, src any /*[]any*/) error {
	v := reflect.Indirect(reflect.ValueOf(dst))
	vsrc := reflect.Indirect(reflect.ValueOf(src))
	switch v.Kind() {
	case reflect.Slice:
		for i := 0; i < vsrc.Len(); i++ {
			itemT := reflect.New(v.Type().Elem()).Interface()
			if err := Map(itemT, vsrc.Index(i).Interface()); err != nil {
				return err
			}
			v = reflect.Append(v, reflect.Indirect(reflect.ValueOf(itemT)))
		}
		reflect.ValueOf(dst).Elem().Set(v)
		return nil
	case reflect.Struct:
		keyfield := -1
		valuefield := -1
		if vsrc.Len() == 0 {
			return nil
		}
		for ix := 0; ix < vsrc.Type().Elem().NumField(); ix++ {
			if vsrc.Type().Elem().Field(ix).Tag.Get("mapper") == "key" {
				keyfield = ix
			} else if vsrc.Type().Elem().Field(ix).Tag.Get("mapper") == "value" {
				valuefield = ix
			}
		}
		if keyfield == -1 || valuefield == -1 {
			return fmt.Errorf("(fromStruct) Unsupported destination type %T for source %T\n Only slice with key/value struct with tags mapper:key and mapper:value are accepted from struct", src, dst)
		}
		for i := 0; i < vsrc.Len(); i++ {
			key := vsrc.Index(i).Field(keyfield).Interface().(string)
			value := redirectValue(vsrc.Index(i).Field(valuefield))
			if value.CanConvert(v.FieldByName(key).Type()) {
				v.FieldByName(key).Set(value.Convert(v.FieldByName(key).Type()))
			} else {
				err := Map(v.FieldByName(key).Addr().Interface(), value.Interface())
				if err != nil {
					return err
				}
			}
		}
		return nil
	default:
		return fmt.Errorf("(fromSlice) Unsupported item type %T for destination %T", src, dst)
	}
}

func fromStruct[T any](dst T, src any) error {
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
				err := Map(itemT, redirectValue(x).Interface())
				if err != nil {
					return err
				}
				nvt.Set(reflect.ValueOf(itemT).Elem())
			}
		}
		return nil
	case reflect.Slice:
		keyfield := -1
		valuefield := -1
		for ix := 0; ix < v.Type().Elem().NumField(); ix++ {
			if v.Type().Elem().Field(ix).Tag.Get("mapper") == "key" {
				keyfield = ix
			} else if v.Type().Elem().Field(ix).Tag.Get("mapper") == "value" {
				valuefield = ix
			}
		}
		if keyfield == -1 || valuefield == -1 {
			return fmt.Errorf("(fromStruct) Unsupported destination type %T for source %T\n Only slice with key/value struct with tags mapper:key and mapper:value are accepted from struct", src, dst)
		}
		keys := funk.Keys(src)
		for _, key := range keys.([]string) {
			var new = true
			for i := 0; i < v.Len(); i++ {
				if v.Index(i).Field(keyfield).Interface().(string) == key {
					v.Index(i).Field(valuefield).Set(reflect.ValueOf(src).FieldByName(key))
					new = false
					break
				}
			}
			if new {
				itemT := reflect.Indirect(reflect.New(v.Type().Elem()))
				itemT.Field(keyfield).SetString(key)
				itemT.Field(valuefield).Set(reflect.ValueOf(src).FieldByName(key))
				v = reflect.Append(v, itemT)
			}
		}
		reflect.ValueOf(dst).Elem().Set(v)
		return nil
	default:
		return fmt.Errorf("(fromStruct) Unsupported source type: %T for destination %T", src, dst)
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
		val, err := strconv.ParseInt(fmt.Sprintf("%v", src), 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(val)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(fmt.Sprintf("%v", src), 10, v.Type().Bits())
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
