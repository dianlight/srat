package mapper

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/thoas/go-funk"
)

func Map(dst any, src any) error {
	//	pdst := reflect.ValueOf(dst)
	if dst == nil || reflect.ValueOf(dst).Kind() != reflect.Ptr {
		return fmt.Errorf("unsupported destination type: %T", dst)
	}
	if src == nil {
		return nil
	}
	//	vdst := reflect.Indirect(pdst)
	vsrc := reflect.Indirect(reflect.ValueOf(src))

	var mapped = reflect.TypeFor[Mappable]()
	if vsrc.Type().Implements(mapped) {
		return src.(Mappable).To(dst)
	} else {
		v := reflect.ValueOf(src)
		switch reflect.Indirect(v).Kind() {

		//switch s := src.(type) {
		//case map[string]any:
		case reflect.Map:
			return fromMap(dst, src.(map[string]any))
		//case string:
		//	destType := reflect.Indirect(reflect.ValueOf(*dst)).Type()
		//	srcValue := reflect.Indirect(reflect.ValueOf(src))
		//	dst = srcValue.Convert(destType).Interface().(*T)
		//	return nil
		case reflect.Slice:
			return fromSlice(dst, src.([]any))
		default:
			return fromInterface(dst, src)
		}
	}
}

/*
func Map[T any](dst *T, src any) error {
	if dst == nil {
		return fmt.Errorf("unsupported destination type: %T", dst)
	}
	var mapped = reflect.TypeFor[Mappable[T]]()
	if reflect.Indirect(reflect.ValueOf(src)).Type().Implements(mapped) {
		return src.(Mappable[T]).To(dst)
	} else {
		switch s := src.(type) {
		case map[string]any:
			return fromMap(dst, s)
		case string:
			destType := reflect.Indirect(reflect.ValueOf(*dst)).Type()
			srcValue := reflect.Indirect(reflect.ValueOf(src))
			dst = srcValue.Convert(destType).Interface().(*T)
			return nil
		//case []any:
		//return fromSlice(dst, s)
		default:
			return fromInterface(dst, s)
		}
	}
}
*/

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
	//	switch d := (interface{}(&dst)).(type) {
	//	case *[]any:
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
		//	case *map[string]any:
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
		//log.Println(keys)
		for _, key := range keys.([]string) {
			//nvt := reflect.Indirect(reflect.ValueOf(dst)).FieldByName(key)
			nvt := redirectValue(reflect.ValueOf(dst)).FieldByName(key)
			//nv := nvt.Interface()
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
		return fmt.Errorf("(fromInterface) Unsupported source type: %T for destination %T", src, dst)
	}
}
