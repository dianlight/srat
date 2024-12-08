package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"

	"dario.cat/mergo"
	"github.com/gorilla/mux"
)

/*
var (
	usersQueue      = map[string](chan *[]User){}
	usersQueueMutex = sync.RWMutex{}
)
*/

// ListUsers godoc
//
//	@Summary		List all configured users
//	@Description	List all configured users
//	@Tags			user
//
// _Accept       json
//
//	@Produce		json
//
// _Param        id   path      int  true  "Account ID"
//
//	@Success		200	{object}	[]User
//
// _Failure      400  {object}  ResponseError
//
//	@Failure		405	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/users [get]
func listUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	jsonResponse, jsonError := json.Marshal(config.OtherUsers)

	if jsonError != nil {
		fmt.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}

}

// GetAdminUser godoc
//
//	@Summary		Get the admin user
//	@Description	get the admin user
//	@Tags			user
//
//
//	@Produce		json
//	@Success		200	{object}	User
//	@Failure		405	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/admin/user [get]
func getAdminUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	jsonResponse, jsonError := json.Marshal(User{
		Username: config.Username,
		Password: config.Password,
	})

	if jsonError != nil {
		fmt.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}

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
//	@Success		200			{object}	User
//	@Failure		405			{object}	ResponseError
//	@Failure		500			{object}	ResponseError
//	@Router			/user/{username} [get]
func getUser(w http.ResponseWriter, r *http.Request) {
	user := mux.Vars(r)["username"]
	w.Header().Set("Content-Type", "application/json")

	index := slices.IndexFunc(config.OtherUsers, func(u User) bool { return u.Username == user })
	if index == -1 {
		w.WriteHeader(http.StatusNotFound)
	} else {
		jsonResponse, jsonError := json.Marshal(config.OtherUsers[index])

		if jsonError != nil {
			log.Println("Unable to encode JSON")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(jsonError.Error()))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(jsonResponse)
		}

	}

}

// CreateUser godoc
//
//	@Summary		Create a user
//	@Description	create e new user
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			user	body		User	true	"Create model"
//	@Success		201		{object}	User
//	@Failure		400		{object}	ResponseError
//	@Failure		405		{object}	ResponseError
//	@Failure		409		{object}	ResponseError
//	@Failure		500		{object}	ResponseError
//	@Router			/user [post]
func createUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var user User

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	index := slices.IndexFunc(config.OtherUsers, func(u User) bool { return u.Username == user.Username })
	if index != -1 {
		w.WriteHeader(http.StatusConflict)
		jsonResponse, jsonError := json.Marshal(ResponseError{Error: "User already exists", Body: config.OtherUsers[index]})

		if jsonError != nil {
			log.Println("Unable to encode JSON")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(jsonError.Error()))
		} else {
			w.Write(jsonResponse)
		}
	} else {

		// TODO: Check the new username with admin username

		config.OtherUsers = append(config.OtherUsers, user)

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

//func notifyClient() {
//	sharesQueueMutex.RLock()
//	for _, v := range sharesQueue {
//		v <- &config.Shares
//	}
//	sharesQueueMutex.RUnlock()
//}

// UpdateUser godoc
//
//	@Summary		Update a user
//	@Description	update e user
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			username	path		string	true	"Name"
//	@Param			user		body		User	true	"Update model"
//	@Success		200			{object}	User
//	@Failure		400			{object}	ResponseError
//	@Failure		405			{object}	ResponseError
//	@Failure		404			{object}	ResponseError
//	@Failure		500			{object}	ResponseError
//	@Router			/user/{username} [put]
//	@Router			/user/{username} [patch]
func updateUser(w http.ResponseWriter, r *http.Request) {
	user := mux.Vars(r)["username"]
	w.Header().Set("Content-Type", "application/json")

	index := slices.IndexFunc(config.OtherUsers, func(u User) bool { return u.Username == user })
	if index == -1 {
		w.WriteHeader(http.StatusNotFound)
	} else {
		var user User

		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mergo.MapWithOverwrite(&config.OtherUsers[index], user)

		//notifyClient()

		jsonResponse, jsonError := json.Marshal(user)

		if jsonError != nil {
			log.Println("Unable to encode JSON")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(jsonError.Error()))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(jsonResponse)
		}

	}

}

// UpdateAdminUser godoc
//
//	@Summary		Update admin user
//	@Description	update admin user
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			user	body		User	true	"Update model"
//	@Success		200		{object}	User
//	@Failure		400		{object}	ResponseError
//	@Failure		405		{object}	ResponseError
//	@Failure		404		{object}	ResponseError
//	@Failure		500		{object}	ResponseError
//	@Router			/admin/user [put]
//	@Router			/admin/user [patch]
func updateAdminUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var user User

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if user.Username != "" {
		config.Username = user.Username
	}
	if user.Password != "" {
		config.Password = user.Password
	}

	//notifyClient()

	jsonResponse, jsonError := json.Marshal(user)

	if jsonError != nil {
		log.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}

}

// DeleteUser godoc
//
//	@Summary		Delete a user
//	@Description	delete a user
//	@Tags			user
//
//	@Param			username	path	string	true	"Name"
//	@Success		204
//	@Failure		400	{object}	ResponseError
//	@Failure		405	{object}	ResponseError
//	@Failure		404	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/user/{username} [delete]
func deleteUser(w http.ResponseWriter, r *http.Request) {
	user := mux.Vars(r)["username"]
	w.Header().Set("Content-Type", "application/json")

	index := slices.IndexFunc(config.OtherUsers, func(u User) bool { return u.Username == user })
	if index == -1 {
		w.WriteHeader(http.StatusNotFound)
	} else {

		config.OtherUsers = slices.Delete(config.OtherUsers, index, index+1)

		//notifyClient()

		w.WriteHeader(http.StatusNoContent)

	}

}

/*

func UsersWsHandler(request WebSocketMessageEnvelope, c chan *WebSocketMessageEnvelope) {
	sharesQueueMutex.Lock()
	if sharesQueue[request.Uid] == nil {
		sharesQueue[request.Uid] = make(chan *Users, 10)
	}
	sharesQueue[request.Uid] <- &config.Shares
	var queue = sharesQueue[request.Uid]
	sharesQueueMutex.Unlock()
	log.Printf("Handle recv: %s %s %d", request.Event, request.Uid, len(sharesQueue))
	for {
		smessage := &WebSocketMessageEnvelope{
			Event: "shares",
			Uid:   request.Uid,
			Data:  <-queue,
		}
		log.Printf("Handle send: %s %s %d", smessage.Event, smessage.Uid, len(c))
		c <- smessage
	}
}

*/
