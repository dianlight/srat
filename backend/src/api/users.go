package api

import (
	"errors"
	"net/http"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
	"github.com/xorcare/pointer"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

type UserHandler struct {
	//ctx               context.Context
	//broascasting      service.BroadcasterServiceInterface
	//volumesQueueMutex sync.RWMutex
	apiContext   *dto.ContextState
	dirtyservice service.DirtyDataServiceInterface
}

func NewUserHandler(apiContext *dto.ContextState, dirtyservice service.DirtyDataServiceInterface) *UserHandler {
	p := new(UserHandler)
	p.apiContext = apiContext
	p.dirtyservice = dirtyservice

	return p
}

func (handler *UserHandler) Routers(srv *fuego.Server) error {
	fuego.Get(srv, "/users", handler.ListUsers, option.Description("List all configured users"), option.Tags("user"))
	fuego.Get(srv, "/useradmin", handler.GetAdminUser, option.Description("Get the admin user"), option.Tags("user"))
	fuego.Put(srv, "/useradmin", handler.UpdateAdminUser, option.Description("Update admin user"), option.Tags("user"))
	fuego.Post(srv, "/user", handler.CreateUser, option.Description("Create a user"), option.Tags("user"))
	fuego.Put(srv, "/user/{username}", handler.UpdateUser, option.Description("Update a user"), option.Tags("user"))
	fuego.Delete(srv, "/user/{username}", handler.DeleteUser, option.Description("Delete a user"), option.Tags("user"))
	return nil
}

func (handler *UserHandler) ListUsers(c fuego.ContextNoBody) ([]dto.User, error) {
	var dbusers dbom.SambaUsers
	err := dbusers.Load()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	var users []dto.User
	var conv converter.DtoToDbomConverterImpl
	for _, dbuser := range dbusers {
		var user dto.User
		err = conv.SambaUserToUser(dbuser, &user)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		if user.IsAdmin == nil {
			user.IsAdmin = pointer.Bool(false)
		}
		users = append(users, user)
	}
	return users, nil
}

func (handler *UserHandler) GetAdminUser(c fuego.ContextNoBody) (*dto.User, error) {
	var adminUser dto.User
	dbUser := dbom.SambaUser{
		IsAdmin: true,
	}
	err := dbUser.GetAdmin()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.SambaUserToUser(dbUser, &adminUser)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	return &adminUser, nil
}

func (handler *UserHandler) CreateUser(c fuego.ContextWithBody[dto.User]) (*dto.User, error) {
	user, err := c.Body()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	var dbUser dbom.SambaUser
	var conv converter.DtoToDbomConverterImpl
	err = conv.UserToSambaUser(user, &dbUser)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	err = dbUser.Create()
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, fuego.ConflictError{
				Title: "User already exists",
			}
		} else {
			return nil, tracerr.Wrap(err)
		}
	}
	handler.dirtyservice.SetDirtyUsers()
	err = conv.SambaUserToUser(dbUser, &user)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	c.SetStatus(http.StatusCreated)
	return &user, nil
}

func (handler *UserHandler) UpdateUser(c fuego.ContextWithBody[dto.User]) (*dto.User, error) {
	username := c.PathParam("username")
	user, err := c.Body()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	dbUser := dbom.SambaUser{
		Username: username,
		IsAdmin:  false,
	}
	err = dbUser.Get()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fuego.NotFoundError{
			Title: "User not found",
		}
	} else if err != nil {
		return nil, tracerr.Wrap(err)
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.UserToSambaUser(user, &dbUser)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	err = dbUser.Save()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	err = conv.SambaUserToUser(dbUser, &user)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	handler.dirtyservice.SetDirtyUsers()

	return &user, nil
}

func (handler *UserHandler) UpdateAdminUser(c fuego.ContextWithBody[dto.User]) (*dto.User, error) {

	user, err := c.Body()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	dbUser := dbom.SambaUser{
		IsAdmin: true,
	}
	err = dbUser.GetAdmin()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.UserToSambaUser(user, &dbUser)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	err = dbUser.Save()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	err = conv.SambaUserToUser(dbUser, &user)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	handler.dirtyservice.SetDirtyUsers()
	return &user, nil
}

func (handler *UserHandler) DeleteUser(c fuego.ContextNoBody) (bool, error) {

	username := c.PathParam("username")

	dbUser := dbom.SambaUser{
		Username: username,
		IsAdmin:  false,
	}
	err := dbUser.Get()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, fuego.NotFoundError{
			Title: "User not found",
		}
	} else if err != nil {
		return false, tracerr.Wrap(err)
	}
	err = dbUser.Delete()
	if err != nil {
		return false, tracerr.Wrap(err)
	}

	handler.dirtyservice.SetDirtyUsers()
	return true, nil
}
