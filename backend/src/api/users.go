package api

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/xorcare/pointer"
	"gorm.io/gorm"
)

type UserHandler struct {
	//ctx               context.Context
	//broascasting      service.BroadcasterServiceInterface
	//volumesQueueMutex sync.RWMutex
	apiContext   *dto.ContextState
	dirtyservice service.DirtyDataServiceInterface
	user_repo    repository.SambaUserRepositoryInterface
}

func NewUserHandler(apiContext *dto.ContextState, dirtyservice service.DirtyDataServiceInterface, user_repo repository.SambaUserRepositoryInterface) *UserHandler {
	p := new(UserHandler)
	p.apiContext = apiContext
	p.dirtyservice = dirtyservice
	p.user_repo = user_repo
	return p
}

func (self *UserHandler) RegisterUserHandler(api huma.API) {
	huma.Get(api, "/users", self.ListUsers, huma.OperationTags("user"))
	huma.Get(api, "/useradmin", self.GetAdminUser, huma.OperationTags("user"))
	huma.Put(api, "/useradmin", self.UpdateAdminUser, huma.OperationTags("user"))
	huma.Post(api, "/user", self.CreateUser, huma.OperationTags("user"))
	huma.Put(api, "/user/{username}", self.UpdateUser, huma.OperationTags("user"))
	huma.Delete(api, "/user/{username}", self.DeleteUser, huma.OperationTags("user"))
}

// ListUsers retrieves a list of users from the database, converts them to DTO format,
// and returns them in the response body. If any error occurs during the process,
// it returns the error.
//
// Parameters:
//   - ctx: The context for the request.
//   - input: An empty struct as input.
//
// Returns:
//   - A struct containing a slice of dto.User in the Body field.
//   - An error if any issue occurs during loading or conversion of users.
func (handler *UserHandler) ListUsers(ctx context.Context, input *struct{}) (*struct{ Body []dto.User }, error) {
	dbusers, err := handler.user_repo.All()
	if err != nil {
		return nil, err
	}
	var users []dto.User
	var conv converter.DtoToDbomConverterImpl
	for _, dbuser := range dbusers {
		var user dto.User
		err = conv.SambaUserToUser(dbuser, &user)
		if err != nil {
			return nil, err
		}
		if user.IsAdmin == nil {
			user.IsAdmin = pointer.Bool(false)
		}
		users = append(users, user)
	}
	return &struct{ Body []dto.User }{Body: users}, nil
}

// GetAdminUser retrieves the admin user from the database and converts it to a DTO user.
// It takes a context and an empty input struct as parameters and returns a struct containing the DTO user or an error.
//
// Parameters:
// - ctx: The context for the request.
// - input: An empty input struct.
//
// Returns:
// - A struct containing the DTO user in the Body field.
// - An error if there is any issue retrieving or converting the user.
func (handler *UserHandler) GetAdminUser(ctx context.Context, input *struct{}) (*struct{ Body dto.User }, error) {
	var adminUser dto.User
	dbUser, err := handler.user_repo.GetAdmin()
	if err != nil {
		return nil, err
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.SambaUserToUser(dbUser, &adminUser)
	if err != nil {
		return nil, err
	}
	return &struct{ Body dto.User }{Body: adminUser}, nil
}

// CreateUser handles the creation of a new user. It takes a context and an input
// struct containing the user data in the form of a dto.User. The function converts
// the dto.User to a dbom.SambaUser, attempts to create the user in the database,
// and handles any errors that occur during this process. If the user already exists,
// it returns a 409 Conflict error. If the creation is successful, it sets the dirty
// users flag, converts the dbom.SambaUser back to a dto.User, and returns the created
// user.
//
// Parameters:
//   - ctx: The context for the request, used for cancellation and deadlines.
//   - input: A struct containing the user data in the form of a dto.User.
//
// Returns:
//   - A struct containing the created user in the form of a dto.User.
//   - An error if the creation fails or if the user already exists.
func (handler *UserHandler) CreateUser(ctx context.Context, input *struct {
	Body dto.User
}) (*struct {
	Status int
	Body   dto.User
}, error) {

	var dbUser dbom.SambaUser
	var conv converter.DtoToDbomConverterImpl
	err := conv.UserToSambaUser(input.Body, &dbUser)
	if err != nil {
		return nil, err
	}
	err = handler.user_repo.Create(&dbUser)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, huma.Error409Conflict("User already exists")
		}
		return nil, err
	}

	var user dto.User
	handler.dirtyservice.SetDirtyUsers()
	err = conv.SambaUserToUser(dbUser, &user)
	if err != nil {
		return nil, err
	}

	return &struct {
		Status int
		Body   dto.User
	}{Status: 201, Body: user}, nil

}

// UpdateUser updates an existing user in the database.
// It retrieves the user by username, updates the user details, and saves the changes.
// If the user is not found, it returns a 404 error.
// If any other error occurs during the process, it returns the error.
// Finally, it marks the user data as dirty.
//
// Parameters:
//   - ctx: The context for the request.
//   - input: A struct containing the username and the user details to be updated.
//
// Returns:
//   - A struct containing the updated user details.
//   - An error if any issue occurs during the update process.
func (handler *UserHandler) UpdateUser(ctx context.Context, input *struct {
	UserName string `path:"username" maxLength:"30" example:"world" doc:"Username"`
	Body     dto.User
}) (*struct{ Body dto.User }, error) {

	dbUser, err := handler.user_repo.GetUserByName(input.UserName)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, huma.Error404NotFound("User not found")
	} else if err != nil {
		return nil, err
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.UserToSambaUser(input.Body, dbUser)
	if err != nil {
		return nil, err
	}

	err = handler.user_repo.Save(dbUser)
	if err != nil {
		return nil, err
	}
	var user dto.User
	err = conv.SambaUserToUser(*dbUser, &user)
	if err != nil {
		return nil, err
	}

	handler.dirtyservice.SetDirtyUsers()

	return &struct{ Body dto.User }{Body: user}, nil
}

// UpdateAdminUser updates the details of an admin user in the system.
// It retrieves the current admin user from the database, converts the input DTO user to the database model,
// saves the updated user back to the database, and then converts the updated database model back to a DTO user.
// Finally, it marks the users as dirty to indicate that they have been updated.
//
// Parameters:
// - ctx: The context for the request, which may include deadlines, cancellation signals, and other request-scoped values.
// - input: A struct containing the user details to be updated.
//
// Returns:
// - A struct containing the updated user details.
// - An error if any operation fails during the update process.
func (handler *UserHandler) UpdateAdminUser(ctx context.Context, input *struct {
	Body dto.User
}) (*struct{ Body dto.User }, error) {

	dbUser, err := handler.user_repo.GetAdmin()
	if err != nil {
		return nil, err
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.UserToSambaUser(input.Body, &dbUser)
	if err != nil {
		return nil, err
	}
	err = handler.user_repo.Save(&dbUser)
	if err != nil {
		return nil, err
	}
	var user dto.User
	err = conv.SambaUserToUser(dbUser, &user)
	if err != nil {
		return nil, err
	}

	handler.dirtyservice.SetDirtyUsers()

	return &struct{ Body dto.User }{Body: user}, nil
}

// DeleteUser godoc
//
//	@Summary		Delete a user
//	@Description	delete a user
//	@Tags			user
//	@Param			username	path	string	true	"Name"
//	@Success		204
//	@Failure		400	{object}	dto.ErrorInfo
//	@Failure		405	{object}	dto.ErrorInfo
//	@Failure		404	{object}	dto.ErrorInfo
//	@Failure		500	{object}	dto.ErrorInfo
//	@Router			/user/{username} [delete]
func (handler *UserHandler) DeleteUser(ctx context.Context, input *struct {
	UserName string `path:"username" maxLength:"30" example:"world" doc:"Username"`
}) (*struct{}, error) {

	err := handler.user_repo.Delete(input.UserName)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, huma.Error404NotFound("User not found")
	} else if err != nil {
		return nil, err
	}

	handler.dirtyservice.SetDirtyUsers()

	return &struct{}{}, nil
}
