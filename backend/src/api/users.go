package api

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"

	"dario.cat/mergo"
	"github.com/dianlight/srat/config"
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

	var users dto.Users

	addon_config := r.Context().Value("addon_config").(*config.Config)
	users.From(addon_config.OtherUsers)
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

	var user dto.User
	addon_config := r.Context().Value("addon_config").(*config.Config)
	user.Username = addon_config.Username
	user.Password = addon_config.Password

	user.ToResponse(http.StatusOK, w)

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
	user := mux.Vars(r)["username"]
	w.Header().Set("Content-Type", "application/json")

	addon_config := r.Context().Value("addon_config").(*config.Config)

	index := slices.IndexFunc(addon_config.OtherUsers, func(u config.User) bool { return u.Username == user })
	if index == -1 {
		w.WriteHeader(http.StatusNotFound)
	} else {
		var user dto.User
		user.From(addon_config.OtherUsers[index])
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

	addon_config := r.Context().Value("addon_config").(*config.Config)

	index := slices.IndexFunc(addon_config.OtherUsers, func(u config.User) bool { return u.Username == user.Username })
	if index != -1 {
		dto.ResponseError{}.ToResponseError(http.StatusConflict, w, "User already exists", addon_config.OtherUsers[index])
	} else {

		// FIXME: Check the new username with admin username
		var cuser config.User
		user.To(cuser)

		addon_config.OtherUsers = append(addon_config.OtherUsers, cuser)
		data_dirty_tracker := r.Context().Value("data_dirty_tracker").(*dto.DataDirtyTracker)
		data_dirty_tracker.Users = true

		//notifyClient()

		jsonResponse, jsonError := json.Marshal(user)

		if jsonError != nil {
			log.Println("Unable to encode JSON")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(jsonError.Error()))
		} else {
			w.WriteHeader(http.StatusCreated)
			w.Write(jsonResponse)
		}

	}
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
	user := mux.Vars(r)["username"]
	w.Header().Set("Content-Type", "application/json")

	addon_config := r.Context().Value("addon_config").(*config.Config)
	index := slices.IndexFunc(addon_config.OtherUsers, func(u config.User) bool { return u.Username == user })
	if index == -1 {
		dto.ResponseError{}.ToResponseError(http.StatusNotFound, w, "User not found", nil)
	} else {
		var user dto.User
		user.FromJSONBody(w, r)

		mergo.MapWithOverwrite(&addon_config.OtherUsers[index], user)
		data_dirty_tracker := r.Context().Value("data_dirty_tracker").(*dto.DataDirtyTracker)
		data_dirty_tracker.Users = true

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
	addon_config := r.Context().Value("addon_config").(*config.Config)

	if user.Username != "" {
		addon_config.Username = user.Username
	}
	if user.Password != "" {
		addon_config.Password = user.Password
	}

	data_dirty_tracker := r.Context().Value("data_dirty_tracker").(*dto.DataDirtyTracker)
	data_dirty_tracker.Users = true

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
	user := mux.Vars(r)["username"]
	w.Header().Set("Content-Type", "application/json")

	addon_config := r.Context().Value("addon_config").(*config.Config)
	index := slices.IndexFunc(addon_config.OtherUsers, func(u config.User) bool { return u.Username == user })
	if index == -1 {
		w.WriteHeader(http.StatusNotFound)
	} else {

		addon_config.OtherUsers = slices.Delete(addon_config.OtherUsers, index, index+1)
		data_dirty_tracker := r.Context().Value("data_dirty_tracker").(*dto.DataDirtyTracker)
		data_dirty_tracker.Users = true

		w.WriteHeader(http.StatusNoContent)

	}

}
