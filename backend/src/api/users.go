package api

import (
	"errors"
	"net/http"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/server"
	"github.com/gorilla/mux"
	"github.com/xorcare/pointer"
	"gorm.io/gorm"
)

type UserHandler struct {
	//ctx               context.Context
	//broascasting      service.BroadcasterServiceInterface
	//volumesQueueMutex sync.RWMutex
	apiContext *ContextState
}

func NewUserHandler(apiContext *ContextState) *UserHandler {
	p := new(UserHandler)
	p.apiContext = apiContext
	//p.ctx = ctx
	//p.broascasting = broascasting
	//p.volumesQueueMutex = sync.RWMutex{}
	return p
}

func (handler *UserHandler) Patterns() []server.RouteDetail {
	return []server.RouteDetail{
		{Pattern: "/users", Method: "GET", Handler: handler.ListUsers},
		{Pattern: "/useradmin", Method: "GET", Handler: handler.GetAdminUser},
		{Pattern: "/useradmin", Method: "PUT", Handler: handler.UpdateAdminUser},
		{Pattern: "/user/{id}", Method: "PUT", Handler: handler.UpdateUser},
		{Pattern: "/user/{id}", Method: "DELETE", Handler: handler.DeleteUser},
	}
}

// ListUsers godoc
//
//	@Summary		List all configured users
//	@Description	List all configured users
//	@Tags			user
//	@Produce		json
//	@Success		200	{object}	[]dto.User
//	@Failure		405	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/users [get]
func (handler *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	var dbusers dbom.SambaUsers
	err := dbusers.Load()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	var users []dto.User
	var conv converter.DtoToDbomConverterImpl
	for _, dbuser := range dbusers {
		var user dto.User
		err = conv.SambaUserToUser(dbuser, &user)
		if err != nil {
			HttpJSONReponse(w, err, nil)
			return
		}
		if user.IsAdmin == nil {
			user.IsAdmin = pointer.Bool(false)
		}
		users = append(users, user)
	}
	//err = mapper.Map(context.Background(), &users, dbusers)
	//if err != nil {
	//	HttpJSONReponse(w, err, nil)
	//	return
	//}
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
//	@Failure		405	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/useradmin [get]
func (handler *UserHandler) GetAdminUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var adminUser dto.User
	dbUser := dbom.SambaUser{
		IsAdmin: true,
	}
	err := dbUser.GetAdmin()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.SambaUserToUser(dbUser, &adminUser)
	//	err = mapper.Map(context.Background(), &adminUser, dbUser)
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
//	@Failure		405			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
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
//	@Failure		400		{object}	ErrorResponse
//	@Failure		405		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/user [post]
func (handler *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {

	var user dto.User
	err := HttpJSONRequest(&user, w, r)
	if err != nil {
		return
	}
	var dbUser dbom.SambaUser
	var conv converter.DtoToDbomConverterImpl
	err = conv.UserToSambaUser(user, &dbUser)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	err = dbUser.Create()
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			HttpJSONReponse(w, errors.New("User already exists"), &Options{
				Code: http.StatusConflict,
			})
		} else {
			HttpJSONReponse(w, err, nil)
		}
		return
	}
	//	context_state := (&dto.Status{}).FromContext(r.Context())
	//context_state := StateFromContext(r.Context())
	handler.apiContext.DataDirtyTracker.Users = true
	err = conv.SambaUserToUser(dbUser, &user)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	HttpJSONReponse(w, user, &Options{
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
//	@Failure		400			{object}	ErrorResponse
//	@Failure		405			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/user/{username} [put]
func (handler *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
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
	var conv converter.DtoToDbomConverterImpl
	err = conv.UserToSambaUser(user, &dbUser)
	//	err = mapper.Map(context.Background(), &dbUser, &user)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	err = dbUser.Save()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	err = conv.SambaUserToUser(dbUser, &user)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	//context_state := (&dto.Status{}).FromContext(r.Context())
	//context_state := StateFromContext(r.Context())
	handler.apiContext.DataDirtyTracker.Users = true
	HttpJSONReponse(w, user, nil)
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
//	@Failure		400		{object}	ErrorResponse
//	@Failure		405		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/useradmin [put]
func (handler *UserHandler) UpdateAdminUser(w http.ResponseWriter, r *http.Request) {

	var user dto.User
	err := HttpJSONRequest(&user, w, r)
	if err != nil {
		return
	}
	dbUser := dbom.SambaUser{
		IsAdmin: true,
	}
	err = dbUser.GetAdmin()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.UserToSambaUser(user, &dbUser)
	//	err = mapper.Map(context.Background(), &dbUser, &user)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	err = dbUser.Save()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	err = conv.SambaUserToUser(dbUser, &user)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	//context_state := (&dto.Status{}).FromContext(r.Context())
	//context_state := StateFromContext(r.Context())
	handler.apiContext.DataDirtyTracker.Users = true
	HttpJSONReponse(w, user, nil)
}

// DeleteUser godoc
//
//	@Summary		Delete a user
//	@Description	delete a user
//	@Tags			user
//	@Param			username	path	string	true	"Name"
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		405	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/user/{username} [delete]
func (handler *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]

	dbUser := dbom.SambaUser{
		Username: username,
		IsAdmin:  false,
	}
	err := dbUser.Get()
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

	//context_state := (&dto.Status{}).FromContext(r.Context())
	//context_state := StateFromContext(r.Context())

	handler.apiContext.DataDirtyTracker.Users = true
	HttpJSONReponse(w, nil, nil)
}
