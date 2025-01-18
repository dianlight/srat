package dto

type Settings struct {
	Workgroup         string        `json:"workgroup,omitempty"`
	Mountoptions      []string      `json:"mountoptions,omitempty"`
	AllowHost         []string      `json:"allow_hosts,omitempty"`
	VetoFiles         []string      `json:"veto_files,omitempty"`
	CompatibilityMode bool          `json:"compatibility_mode,omitempty"`
	EnableRecycleBin  bool          `json:"recyle_bin_enabled,omitempty"`
	Interfaces        []string      `json:"interfaces,omitempty"`
	BindAllInterfaces bool          `json:"bind_all_interfaces,omitempty"`
	LogLevel          string        `json:"log_level,omitempty"`
	MultiChannel      bool          `json:"multi_channel,omitempty"`
	UpdateChannel     UpdateChannel `json:"update_channel,omitempty"`
}

/*
func (self *Settings) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}

func (self *Settings) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self Settings) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}

func (self Settings) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self Settings) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self Settings) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *Settings) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}

func isNil(a interface{}) bool {
	defer func() { recover() }()
	return a == nil || reflect.ValueOf(a).IsNil()
}

func (self Settings) ToArray(dst interface{}) error {
	var keyfield string
	var valuefield string
	elemType := reflect.TypeOf(dst).Elem().Elem()
	for _, afs := range reflect.VisibleFields(elemType) {
		if afs.Tag.Get("from_map") == "key" {
			keyfield = afs.Name
		} else if afs.Tag.Get("from_map") == "value" {
			valuefield = afs.Name
		}
	}
	if keyfield == "" || valuefield == "" {
		return tracerr.Wrap(err)ors.New("missing annotation from_map key field or value field")
	}
	var mapdata map[string]interface{}
	err := self.ToMap(&mapdata)
	if err != nil {
		return tracerr.Wrap(err)
	}
	//fmt.Println(mapdata)
	for key, val := range mapdata {
		newElem := reflect.New(elemType).Elem()
		newElem.FieldByName(keyfield).SetString(key)

		fbvalue := newElem.FieldByName(valuefield)
		if isNil(val) {
			continue
		}
		ve := reflect.ValueOf(val)
		//fmt.Println(">>", val)
		fbvalue.Set(ve.Convert(fbvalue.Type()))

		sliceDst := reflect.Indirect(reflect.ValueOf(dst))

		sliceDst.Set(reflect.Append(sliceDst, newElem))
	}
	return nil
}

func (self *Settings) FromArray(value interface{}) error {
	var keyfield string
	var valuefield string
	elemType := reflect.Indirect(reflect.ValueOf(value)).Type().Elem() //reflect.TypeOf(value).Elem().Elem()
	for _, afs := range reflect.VisibleFields(elemType) {
		if afs.Tag.Get("from_map") == "key" {
			keyfield = afs.Name
		} else if afs.Tag.Get("from_map") == "value" {
			valuefield = afs.Name
		}
	}
	if keyfield == "" || valuefield == "" {
		return tracerr.Wrap(err)ors.New("missing annotation from_map key field or value field")
	}
	selfValue := reflect.Indirect(reflect.ValueOf(self))
	arrValue := reflect.Indirect(reflect.ValueOf(value))
	for i := 0; i < arrValue.Len(); i++ {
		//fmt.Printf("%d %#v %v\n", i, arrValue.Index(i), arrValue.Index(i).FieldByName(keyfield).Interface().(string))
		fbset := selfValue.FieldByName(arrValue.Index(i).FieldByName(keyfield).Interface().(string))
		tovalue := arrValue.Index(i).FieldByName(valuefield).Elem()
		if fbset.IsValid() && fbset.CanSet() {
			if tovalue.CanConvert(fbset.Type()) {
				//fmt.Printf("|--> %#v %v\n", tovalue.Interface(), fbset.Type())
				fbset.Set(tovalue.Convert(fbset.Type()))
			} else if fbset.Type().Kind() == reflect.Slice {
				nslice := reflect.MakeSlice(fbset.Type(), tovalue.Len(), tovalue.Len())
				//nslice.SetPointer(tovalue.UnsafePointer())
				for ix := 0; ix < tovalue.Len(); ix++ {
					nslice.Index(ix).Set(tovalue.Index(ix).Elem().Convert(fbset.Type().Elem()))
					//nslice.Index(ix).Elem().SetPointer(tovalue.Index(ix).UnsafePointer())
				}

				//reflect.Copy(nslice, tovalue)
				fbset.Set(nslice)
				//fmt.Printf("|-s-> %#v %v\n", tovalue.Interface(), fbset.Type())
			} else {
				return tracerr.Wrap(err)ors.New("unsupported type " + fbset.Type().Name())
			}
		} else {
			return tracerr.Wrap(err)ors.New(fmt.Sprintf("Invalid Field %#v %v %v\n", tovalue.Interface(), fbset.Type(), fbset.Type().Kind()))
		}
	}

	return nil
}

func (self *Settings) ToMap(dst *map[string]interface{}) error {
	retmap := make(map[string]interface{})
	selfValue := reflect.Indirect(reflect.ValueOf(self))
	for i, afs := range reflect.VisibleFields(selfValue.Type()) {
		//fmt.Printf("%d *-> %#v\n", i, afs)
		retmap[afs.Name] = selfValue.FieldByIndex([]int{i}).Interface()
	}
	*dst = retmap

	return nil
}
*/
