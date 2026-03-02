package api

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"gitlab.com/tozd/go/errors"
)

type UserHandler struct {
	userService service.UserServiceInterface
}

func NewUserHandler(
	userService service.UserServiceInterface,
) *UserHandler {
	p := new(UserHandler)
	p.userService = userService
	return p
}

func (self *UserHandler) RegisterUserHandler(api huma.API) {
	huma.Get(api, "/users", self.ListUsers, huma.OperationTags("user"))
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
	users, err := handler.userService.ListUsers()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list users")
	}
	return &struct{ Body []dto.User }{Body: users}, nil
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
	Body dto.User `required:"true"`
}) (*struct {
	Status int
	Body   dto.User
}, error) {
	createdUser, err := handler.userService.CreateUser(input.Body)
	if err != nil {
		if errors.Is(err, dto.ErrorUserAlreadyExists) {
			return nil, huma.Error409Conflict(err.Error())
		}
		return nil, errors.Wrapf(err, "failed to create user %s", input.Body.Username)
	}

	return &struct {
		Status int
		Body   dto.User
	}{Status: 201, Body: *createdUser}, nil
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
	UserName string   `path:"username" maxLength:"30" example:"world" doc:"Username"`
	Body     dto.User `required:"true"`
}) (*struct{ Body dto.User }, error) {
	updatedUser, err := handler.userService.UpdateUser(input.UserName, input.Body)
	if err != nil {
		if errors.Is(err, dto.ErrorUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, dto.ErrorUserAlreadyExists) {
			return nil, huma.Error409Conflict(err.Error())
		}
		return nil, errors.Wrapf(err, "failed to update user %s", input.UserName)
	}
	return &struct{ Body dto.User }{Body: *updatedUser}, nil
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
	Body dto.User `required:"true"`
}) (*struct{ Body dto.User }, error) {
	updatedAdmin, err := handler.userService.UpdateAdminUser(input.Body)
	if err != nil {
		if errors.Is(err, dto.ErrorUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, dto.ErrorUserAlreadyExists) {
			return nil, huma.Error409Conflict(err.Error())
		}
		return nil, errors.Wrap(err, "failed to update admin user")
	}
	return &struct{ Body dto.User }{Body: *updatedAdmin}, nil
}

func (handler *UserHandler) DeleteUser(ctx context.Context, input *struct {
	UserName string `path:"username" minimum:"1" maxLength:"30" example:"world" doc:"Username"`
}) (*struct{}, error) {
	err := handler.userService.DeleteUser(input.UserName)
	if err != nil {
		if errors.Is(err, dto.ErrorUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, errors.Wrapf(err, "failed to delete user %s", input.UserName)
	}
	return &struct{}{}, nil
}
