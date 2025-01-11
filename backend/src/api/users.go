package api

import (
	"errors"
	"net/http"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/mapper"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// ListUsers godoc
//
//	@Summary		List all configured users
//	@Description	List all configured users
//	@Tags			user
//	@Produce		json
//	@Success		200	{object}	dto.Users
//	@Failure		405	{object}	dto.ResponseError
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/users [get]
func ListUsers(w http.ResponseWriter, r *http.Request) {
	var dbusers dbom.SambaUsers
	err := dbusers.Load()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	var users []dto.User
	err = mapper.Map(&users, dbusers)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	HttpJSONReponse(w, users, &Options{
		Code: http.StatusOK,
	})
}

// GetAdminUser godoc
//
//	@Summary		Get the admin user
//	@Description	get the admin user
//	@Tags			user
//	@Produce		json
//	@Success		200	{object}	dto.User
//	@Failure		405	{object}	dto.ResponseError
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/admin/user [get]
func GetAdminUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var adminUser dto.User
	dbUser := dbom.SambaUser{
		IsAdmin: true,
	}
	err := dbUser.Get()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	err = mapper.Map(&adminUser, dbUser)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	HttpJSONReponse(w, adminUser, &Options{
		Code: http.StatusOK,
	})
}

// GetUser godoc
//
//	@Summary		Get a user
//	@Description	get user by Name
//	@Tags			user
//
//
//	@Produce		json
//	@Param			username	path		string	true	"Name"
//	@Success		200			{object}	dto.User
//	@Failure		405			{object}	dto.ResponseError
//	@Failure		500			{object}	dto.ResponseError
//	@Router			/user/{username} [get]
/*
func GetUser(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	w.Header().Set("Content-Type", "application/json")

	context_state := (&dto.ContextState{}).FromContext(r.Context())

	user, index := context_state.Users.Get(username)
	if index == -1 {
		w.WriteHeader(http.StatusNotFound)
	} else {
		user.ToResponse(http.StatusOK, w)
	}

}
*/

// CreateUser godoc
//
//	@Summary		Create a user
//	@Description	create e new user
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			user	body		dto.User	true	"Create model"
//	@Success		201		{object}	dto.User
//	@Failure		400		{object}	dto.ResponseError
//	@Failure		405		{object}	dto.ResponseError
//	@Failure		409		{object}	dto.ResponseError
//	@Failure		500		{object}	dto.ResponseError
//	@Router			/user [post]
func CreateUser(w http.ResponseWriter, r *http.Request) {

	var user dto.User
	err := HttpJSONRequest(&user, w, r)
	if err != nil {
		return
	}
	var dbUser dbom.SambaUser
	err = mapper.Map(&dbUser, user)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	err = dbUser.Save()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	context_state := (&dto.ContextState{}).FromContext(r.Context())
	context_state.DataDirtyTracker.Users = true
	HttpJSONReponse(w, nil, &Options{
		Code: http.StatusCreated,
	})
}

// UpdateUser godoc
//
//	@Summary		Update a user
//	@Description	update e user
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			username	path		string		true	"Name"
//	@Param			user		body		dto.User	true	"Update model"
//	@Success		200			{object}	dto.User
//	@Failure		400			{object}	dto.ResponseError
//	@Failure		405			{object}	dto.ResponseError
//	@Failure		404			{object}	dto.ResponseError
//	@Failure		500			{object}	dto.ResponseError
//	@Router			/user/{username} [put]
//	@Router			/user/{username} [patch]
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]

	var user dto.User
	err := HttpJSONRequest(&user, w, r)
	if err != nil {
		return
	}

	dbUser := dbom.SambaUser{
		Username: username,
		IsAdmin:  false,
	}
	err = dbUser.Get()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	err = mapper.Map(&dbUser, &user)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	context_state := (&dto.ContextState{}).FromContext(r.Context())
	context_state.DataDirtyTracker.Users = true
	HttpJSONReponse(w, nil, nil)
}

// UpdateAdminUser godoc
//
//	@Summary		Update admin user
//	@Description	update admin user
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			user	body		dto.User	true	"Update model"
//	@Success		200		{object}	dto.User
//	@Failure		400		{object}	dto.ResponseError
//	@Failure		405		{object}	dto.ResponseError
//	@Failure		404		{object}	dto.ResponseError
//	@Failure		500		{object}	dto.ResponseError
//	@Router			/admin/user [put]
//	@Router			/admin/user [patch]
func UpdateAdminUser(w http.ResponseWriter, r *http.Request) {

	var user dto.User
	err := HttpJSONRequest(&user, w, r)
	if err != nil {
		return
	}
	dbUser := dbom.SambaUser{
		IsAdmin: true,
	}
	err = dbUser.Get()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	err = mapper.Map(&dbUser, &user)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	context_state := (&dto.ContextState{}).FromContext(r.Context())
	context_state.DataDirtyTracker.Users = true
	HttpJSONReponse(w, nil, nil)
}

// DeleteUser godoc
//
//	@Summary		Delete a user
//	@Description	delete a user
//	@Tags			user
//	@Param			username	path	string	true	"Name"
//	@Success		204
//	@Failure		400	{object}	dto.ResponseError
//	@Failure		405	{object}	dto.ResponseError
//	@Failure		404	{object}	dto.ResponseError
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/user/{username} [delete]
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]

	var user dto.User
	err := HttpJSONRequest(&user, w, r)
	if err != nil {
		return
	}
	dbUser := dbom.SambaUser{
		Username: username,
		IsAdmin:  false,
	}
	err = dbUser.Get()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		HttpJSONReponse(w, nil, &Options{
			Code: http.StatusNotFound,
		})
		return
	} else if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	err = dbUser.Delete()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	context_state := (&dto.ContextState{}).FromContext(r.Context())

	context_state.DataDirtyTracker.Users = true
	HttpJSONReponse(w, nil, nil)
}
