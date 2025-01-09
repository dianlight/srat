package dto

import (
	"net/http"
	"slices"

	"github.com/jinzhu/copier"
)

type Users []User

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

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
