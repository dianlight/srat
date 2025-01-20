package dto

//type Users []User

type User struct {
	Username *string `json:"username"`
	Password *string `json:"password,omitempty"`
	IsAdmin  *bool   `json:"is_admin,omitempty"`
}

/*

func (m User) To(ctx context.Context, dst any) (bool, error) {
	switch v := dst.(type) {
	case *string:
		*v = *m.Username
		return true, nil
	default:
		return false, nil
	}
}

func (m *User) From(ctx context.Context, src any) (bool, error) {
	switch v := src.(type) {
	case string:
		stack := ctx.Value("mapper_stack").(*[]mapper.Stack)
		users := (*((*stack)[0].Dst.(*SharedResource))).Users
		nv := funk.Find(users, func(u User) bool { return u.Username == &v })
		if nv != nil {
			*m = nv.(User)
			return true, nil
		}
		fmt.Printf("User not in %+v found: %+v", users, v)
		return true, nil
	default:
		return false, nil
	}
}
*/

/*
func (self *User) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *User) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self User) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self User) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self User) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self User) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *User) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}

func (self *Users) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *Users) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self Users) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self Users) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self Users) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self Users) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *Users) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}

func (self *Users) Get(username string) (*User, int) {
	index := slices.IndexFunc(*self, func(u User) bool { return u.Username == username })
	if index == -1 {
		return nil, index
	} else {
		return &(*self)[index], index
	}
}

func (self Users) Users() (Users, error) {
	tmp := slices.Clone(self)
	result := slices.DeleteFunc(tmp, func(u User) bool { return u.IsAdmin })
	return result, nil
}

func (self Users) AdminUser() (*User, error) {
	tmp := slices.Clone(self)
	result := slices.DeleteFunc(tmp, func(u User) bool { return !u.IsAdmin })
	if len(result) == 0 {
		return nil, nil
	} else {
		return &result[0], nil
	}
}
*/
