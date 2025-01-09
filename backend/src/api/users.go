package api

import (
	"errors"
	"net/http"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
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
	w.Header().Set("Content-Type", "application/json")

	var dbusers dbom.SambaUsers
	err := dbusers.Load()
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	var users dto.Users
	err = users.From(dbusers)
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	users.ToResponse(http.StatusOK, w)
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
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	err = adminUser.From(&dbUser)
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	adminUser.ToResponse(http.StatusOK, w)
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
	w.Header().Set("Content-Type", "application/json")

	var user dto.User
	user.FromJSONBody(w, r)
	var dbUser dbom.SambaUser
	err := user.To(&dbUser)
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusBadRequest, w, "Invalid user data", user)
		return
	}
	err = dbUser.Save()
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	user.ToResponse(http.StatusCreated, w)

	context_state := (&dto.ContextState{}).FromContext(r.Context())
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

	var user dto.User
	user.FromJSONBody(w, r)
	dbUser := dbom.SambaUser{
		Username: username,
		IsAdmin:  false,
	}
	err := dbUser.Get()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	user.ToIgnoreEmpty(dbUser)

	context_state := (&dto.ContextState{}).FromContext(r.Context())
	context_state.DataDirtyTracker.Users = true
	user.ToResponse(http.StatusOK, w)
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
	dbUser := dbom.SambaUser{
		IsAdmin: true,
	}
	err := dbUser.Get()
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	user.ToIgnoreEmpty(dbUser)

	context_state := (&dto.ContextState{}).FromContext(r.Context())
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

	var user dto.User
	user.FromJSONBody(w, r)
	dbUser := dbom.SambaUser{
		Username: username,
		IsAdmin:  false,
	}
	err := dbUser.Get()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	err = dbUser.Delete()
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}

	context_state := (&dto.ContextState{}).FromContext(r.Context())

	context_state.DataDirtyTracker.Users = true
	w.WriteHeader(http.StatusNoContent)

}
