package mapper

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/thoas/go-funk"
	"github.com/ztrue/tracerr"
)

func Map(dst any, src any) error {
	vdst := reflect.ValueOf(dst)
	if vdst.Kind() != reflect.Ptr {
		return tracerr.Errorf("Unsupported destination type: %T a pointer is required!", dst)
	}
	if vdst.IsNil() {
		return tracerr.Errorf("Unsupported Nil destination")
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
			return tracerr.Wrap(err)
		}
		if match {
			return nil
		}
	} else if val, ok := dst.(MappableFrom); ok {
		match, err := val.From(src)
		if err != nil {
			return tracerr.Wrap(err)
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
		return tracerr.Errorf("Unsupported source type: %T for destination %T", src, dst)
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
				return tracerr.Wrap(err)
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
					return tracerr.Wrap(err)
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
					return tracerr.Wrap(err)
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
				return tracerr.Wrap(err)
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
			return tracerr.Errorf("(fromSlice) Unsupported destination type %T for source %T\n Only slice of key/value struct with tags mapper:key and mapper:value are accepted as source of struct", src, dst)
		}
		for i := 0; i < vsrc.Len(); i++ {
			key := vsrc.Index(i).Field(keyfield).Interface().(string)
			value := redirectValue(vsrc.Index(i).Field(valuefield))
			if value.CanConvert(v.FieldByName(key).Type()) {
				v.FieldByName(key).Set(value.Convert(v.FieldByName(key).Type()))
			} else {
				itemT := v.FieldByName(key).Addr()
				if v.FieldByName(key).Type().Kind() == reflect.Ptr {
					itemT = v.FieldByName(key)
				}
				if itemT.IsNil() {
					itemT.Set(reflect.New(value.Type()))
				}
				err := Map(itemT.Interface(), value.Interface())
				if err != nil {
					return tracerr.Wrap(err)
				}
			}
		}
		return nil
	case reflect.Map:
		keyfield := -1
		if vsrc.Len() == 0 {
			return nil
		}
		for ix := 0; ix < vsrc.Type().Elem().NumField(); ix++ {
			if vsrc.Type().Elem().Field(ix).Tag.Get("mapper") == "mapkey" {
				keyfield = ix
				break
			}
		}
		if keyfield == -1 {
			return tracerr.Errorf("(fromSlice) Unsupported destination type %T for source %T\n Only slice of struct with tags mapper:mapkey are accepted as source for map", src, dst)
		}
		for i := 0; i < vsrc.Len(); i++ {
			itemT := reflect.New(v.Type().Elem()).Interface()
			if err := Map(itemT, vsrc.Index(i).Interface()); err != nil {
				return tracerr.Wrap(err)
			}
			v.SetMapIndex(vsrc.Index(i).Field(keyfield), reflect.ValueOf(itemT).Elem())
		}
		reflect.ValueOf(dst).Elem().Set(v)
		return nil
	default:
		return tracerr.Errorf("(fromSlice) Unsupported item type %T for destination %T", src, dst)
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
					return tracerr.Wrap(err)
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
			return tracerr.Errorf("(fromStruct) Unsupported destination type %T for source %T\n Only slice with key/value struct with tags mapper:key and mapper:value are accepted from struct", src, dst)
		}
		keys := funk.Keys(src)
		for _, key := range keys.([]string) {
			newvalue := reflect.ValueOf(src).FieldByName(key)
			if newvalue.IsZero() {
				continue
			}
			var new = true
			for i := 0; i < v.Len(); i++ {
				if v.Index(i).Field(keyfield).Interface().(string) == key {
					v.Index(i).Field(valuefield).Set(newvalue)
					new = false
					break
				}
			}
			if new {
				itemT := reflect.Indirect(reflect.New(v.Type().Elem()))
				itemT.Field(keyfield).SetString(key)
				itemT.Field(valuefield).Set(newvalue)
				v = reflect.Append(v, itemT)
			}
		}
		reflect.ValueOf(dst).Elem().Set(v)
		return nil
	default:
		return tracerr.Errorf("(fromStruct) Unsupported source type: %T for destination %T", src, dst)
	}
}

func fromNative[T any](dst T, src any) error {
	return _fromNative(reflect.ValueOf(dst), reflect.ValueOf(src))
}

func _fromNative(v reflect.Value, src reflect.Value) error {
	//v := /*reflect.Indirect(*/ reflect.ValueOf(dst) /*)*/
	//v := redirectValue(reflect.ValueOf(dst))
	switch v.Kind() {
	case reflect.Float64, reflect.Float32:
		val, err := strconv.ParseFloat(fmt.Sprintf("%v", src.Interface()), v.Type().Bits())
		if err != nil {
			return tracerr.Wrap(err)
		}
		reflect.Indirect(v).SetFloat(val)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(fmt.Sprintf("%v", src.Interface()), 10, v.Type().Bits())
		if err != nil {
			return tracerr.Wrap(err)
		}
		reflect.Indirect(v).SetInt(val)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(fmt.Sprintf("%v", src.Interface()), 10, v.Type().Bits())
		if err != nil {
			return tracerr.Wrap(err)
		}
		reflect.Indirect(v).SetUint(val)
		return nil
	case reflect.String:
		reflect.Indirect(v).SetString(fmt.Sprintf("%v", src.Interface()))
		return nil
	case reflect.Bool:
		val, err := strconv.ParseBool(fmt.Sprintf("%v", src.Interface()))
		if err != nil {
			return tracerr.Wrap(err)
		}
		reflect.Indirect(v).SetBool(val)
		return nil
	case reflect.Ptr:
		if src.IsZero() {
			v.Set(reflect.Zero(v.Type().Elem()))
			return nil
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return _fromNative(reflect.Indirect(v), src)
	default:
		return tracerr.Errorf("(fromNative) Unsupported source type: %s for destination %s", src.Type().Name(), v.Type().Name())
	}
}
