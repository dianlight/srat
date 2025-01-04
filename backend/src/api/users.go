package api

import (
	"net/http"
	"slices"

	"dario.cat/mergo"
	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
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
	w.Header().Set("Content-Type", "application/json")

	context_state := (&dto.ContextState{}).FromContext(r.Context())
	context_state.Users.ToResponse(http.StatusOK, w)
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

	context_state := (&dto.ContextState{}).FromContext(r.Context())
	context_state.AdminUser.ToResponse(http.StatusOK, w)
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
	w.Header().Set("Content-Type", "application/json")

	var user dto.User
	user.FromJSONBody(w, r)

	context_state := (&dto.ContextState{}).FromContext(r.Context())
	_, index := context_state.Users.Get(user.Username)
	if index != -1 {
		dto.ResponseError{}.ToResponseError(http.StatusConflict, w, "User already exists", user)
		return
	}

	if user.Username == context_state.AdminUser.Username {
		dto.ResponseError{}.ToResponseError(http.StatusConflict, w, "User already exists", user)
		return
	}

	context_state.Users = append(context_state.Users, user)
	context_state.DataDirtyTracker.Users = true

	user.ToResponse(http.StatusCreated, w)
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
	w.Header().Set("Content-Type", "application/json")

	context_state := (&dto.ContextState{}).FromContext(r.Context())
	user, index := context_state.Users.Get(username)
	if index == -1 {
		dto.ResponseError{}.ToResponseError(http.StatusNotFound, w, "User not found", nil)
	} else {
		var nuser dto.User
		nuser.FromJSONBody(w, r)
		mergo.Map(user, nuser, mergo.WithOverride)
		context_state.DataDirtyTracker.Users = true
		user.ToResponse(http.StatusOK, w)
	}

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
	w.Header().Set("Content-Type", "application/json")

	var user dto.User
	user.FromJSONBody(w, r)
	context_state := (&dto.ContextState{}).FromContext(r.Context())

	if user.Username != "" {
		context_state.AdminUser.Username = user.Username
	}
	if user.Password != "" {
		context_state.AdminUser.Password = user.Password
	}
	context_state.DataDirtyTracker.Users = true
	user.ToResponse(http.StatusOK, w)
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
	w.Header().Set("Content-Type", "application/json")

	context_state := (&dto.ContextState{}).FromContext(r.Context())
	_, index := context_state.Users.Get(username)
	if index == -1 {
		w.WriteHeader(http.StatusNotFound)
	} else {
		context_state.Users = slices.Delete(context_state.Users, index, index+1)
		context_state.DataDirtyTracker.Users = true
		w.WriteHeader(http.StatusNoContent)
	}

}
