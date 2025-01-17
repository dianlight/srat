package dto

type HAMountUsage string // https://developers.home-assistant.io/docs/api/supervisor/models#mount

const (
	UsageAsBackup HAMountUsage = "backup"
	UsageAsMedia  HAMountUsage = "media"
	UsageAsShare  HAMountUsage = "share"
	UsageAsNone   HAMountUsage = "none"
)

type SharedResource struct {
	ID          *uint        `json:"id,omitempty"`
	Name        string       `json:"name,omitempty"  mapper:"mapkey"`
	Path        string       `json:"path"`
	FS          string       `json:"fs"`
	Disabled    bool         `json:"disabled,omitempty"`
	Users       []User       `json:"users"`
	RoUsers     []User       `json:"ro_users"`
	TimeMachine bool         `json:"timemachine,omitempty"`
	Usage       HAMountUsage `json:"usage,omitempty"`

	//	DirtyStatus bool    `json:"id_dirty,omitempty"`
	DeviceId *uint64 `json:"device_id,omitempty"`
	Invalid  bool    `json:"invalid,omitempty"`
}

/*

func (self *SharedResource) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *SharedResource) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self SharedResource) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self SharedResource) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self SharedResource) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self SharedResource) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *SharedResource) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}

type SharedResources map[string]SharedResource

func (self SharedResources) Get(key string) (sharedResource *SharedResource, found bool) {
	sharedResource_, ok := self[key]
	if ok {
		return &sharedResource_, true
	}
	return nil, false
}

func (self *SharedResources) From(value interface{}) error {
	//	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
	var rerr error = nil
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			rerr = err.(error)
		}
	}()
	*self = SharedResources(funk.Map(value, func(k string, v any) (string, SharedResource) {
		var dst SharedResource
		funk.ForEach(funk.Keys(&SharedResource{}), func(k any) {
			if *k.(*string) == "Users" || *k.(*string) == "RoUsers" {
				var users Users
				err := copier.Copy(&users, v)
				if err != nil {
					panic(err)
				}
				funk.Set(&dst, users, *k.(*string))
				//dst.Users = users
			} else if *k.(*string) == "Usage" {
				dst.Usage = HAMountUsage(funk.Get(v, *k.(*string)).(string))
			} else {
				funk.Set(&dst, funk.Get(v, *k.(*string)), *k.(*string))
			}
		})
		return k, dst
	}).(map[string]SharedResource))
	return rerr
}
func (self *SharedResources) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self SharedResources) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}

func (self SharedResources) ToArray(value interface{}) error {
	vals := slices.Collect(maps.Values(self))
	return copier.CopyWithOption(value, vals, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}

func (self *SharedResources) FromArray(value interface{}, keyfield string) error {
	if value == nil {
		return errors.New("Missing array in the request body")
	}
	if reflect.Indirect(reflect.ValueOf(value)).Type().Kind() != reflect.Slice {
		return errors.New("Expected array in the request body")
	}
	arrValue := reflect.Indirect(reflect.ValueOf(value))
	for i := 0; i < arrValue.Len(); i++ {
		shareName := reflect.Indirect(arrValue.Index(i)).FieldByName(keyfield).Interface().(string)
		if shareName == "" {
			return errors.New("Missing '" + keyfield + "' field in the array item")
		}
		var toShare SharedResource
		copier.CopyWithOption(&toShare, arrValue.Index(i).Interface(), copier.Option{DeepCopy: true})
		if isNil(*self) {
			*self = make(SharedResources)
		}
		(*self)[shareName] = toShare
	}
	/*
		for _, v := range reflect.Indirect(reflect.ValueOf(value)).Interface().([]interface{}) {
			shareName := reflect.Indirect(reflect.ValueOf(v)).FieldByName(keyfield).Interface().(string)
			if shareName == "" {
				return errors.New("Missing '" + keyfield + "' field in the array item")
			}
			var toShare SharedResource
			copier.CopyWithOption(&toShare, v, copier.Option{DeepCopy: true})
			self[shareName] = toShare
		}
	* /
	return nil
}

func (self SharedResources) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self SharedResources) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self SharedResources) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *SharedResources) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}

*/
