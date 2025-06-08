package service

import (
	"log/slog"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type UserServiceInterface interface {
	ListUsers() ([]dto.User, error)
	CreateUser(user dto.User) (*dto.User, error)
	UpdateUser(currentUsername string, userDto dto.User) (*dto.User, error)
	UpdateAdminUser(userDto dto.User) (*dto.User, error)
	DeleteUser(username string) error
}

type UserService struct {
	userRepo     repository.SambaUserRepositoryInterface
	dirtyService DirtyDataServiceInterface
	shareService ShareServiceInterface
}

type UserServiceParams struct {
	fx.In
	UserRepo     repository.SambaUserRepositoryInterface
	DirtyService DirtyDataServiceInterface
	ShareService ShareServiceInterface
}

func NewUserService(params UserServiceParams) UserServiceInterface {
	return &UserService{
		userRepo:     params.UserRepo,
		dirtyService: params.DirtyService,
		shareService: params.ShareService,
	}
}

func (s *UserService) ListUsers() ([]dto.User, error) {
	dbusers, err := s.userRepo.All()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list users from repository")
	}
	var users []dto.User
	var conv converter.DtoToDbomConverterImpl
	for _, dbuser := range dbusers {
		var user dto.User
		err = conv.SambaUserToUser(dbuser, &user)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert db user %s to dto", dbuser.Username)
		}
		users = append(users, user)
	}
	return users, nil
}

func (s *UserService) CreateUser(userDto dto.User) (*dto.User, error) {
	var dbUser dbom.SambaUser
	var conv converter.DtoToDbomConverterImpl
	if err := conv.UserToSambaUser(userDto, &dbUser); err != nil {
		return nil, errors.Wrap(err, "failed to convert user DTO to DBOM")
	}

	slog.Debug("Attempting to create user in DB", "dbUser", dbUser)
	if err := s.userRepo.Create(&dbUser); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, dto.ErrorUserAlreadyExists
		}
		slog.Error("Error creating user in repository", "err", err)
		return nil, errors.Wrap(err, "failed to create user in repository")
	}

	s.dirtyService.SetDirtyUsers()
	go s.shareService.NotifyClient()

	var createdUserDto dto.User
	if err := conv.SambaUserToUser(dbUser, &createdUserDto); err != nil {
		return nil, errors.Wrap(err, "failed to convert created DBOM user back to DTO")
	}
	return &createdUserDto, nil
}

func (s *UserService) UpdateUser(currentUsername string, userDto dto.User) (*dto.User, error) {
	dbUser, err := s.userRepo.GetUserByName(currentUsername)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, dto.ErrorUserNotFound
		}
		return nil, errors.Wrapf(err, "failed to get user %s from repository", currentUsername)
	}

	var conv converter.DtoToDbomConverterImpl

	if userDto.Username != "" && userDto.Username != currentUsername {
		// Handle rename
		if _, checkErr := s.userRepo.GetUserByName(userDto.Username); checkErr == nil {
			return nil, errors.WithMessagef(dto.ErrorUserAlreadyExists, "cannot rename to %s, user already exists", userDto.Username)
		} else if !errors.Is(checkErr, gorm.ErrRecordNotFound) {
			return nil, errors.Wrapf(checkErr, "error checking if new username %s exists", userDto.Username)
		}
		if err := s.userRepo.Rename(currentUsername, userDto.Username); err != nil {
			return nil, errors.Wrapf(err, "failed to rename user from %s to %s", currentUsername, userDto.Username)
		}
		dbUser.Username = userDto.Username // Update username in the struct for subsequent save
	}

	// Apply other DTO changes to dbUser
	// Ensure the DTO's username is consistent with dbUser.Username for the converter
	originalDtoUsername := userDto.Username
	userDto.Username = dbUser.Username
	if err := conv.UserToSambaUser(userDto, dbUser); err != nil {
		userDto.Username = originalDtoUsername // Restore for error message consistency
		return nil, errors.Wrap(err, "failed to convert user DTO to DBOM for update")
	}
	userDto.Username = originalDtoUsername // Restore for caller, if they inspect it

	if err := s.userRepo.Save(dbUser); err != nil {
		return nil, errors.Wrapf(err, "failed to save updated user %s to repository", dbUser.Username)
	}

	s.dirtyService.SetDirtyUsers()
	go s.shareService.NotifyClient()

	var updatedUserDto dto.User
	if err := conv.SambaUserToUser(*dbUser, &updatedUserDto); err != nil {
		return nil, errors.Wrap(err, "failed to convert updated DBOM user back to DTO")
	}
	return &updatedUserDto, nil
}

func (s *UserService) UpdateAdminUser(userDto dto.User) (*dto.User, error) {
	// This method is more complex due to potential admin username change.
	// The existing logic in api.UserHandler for UpdateAdminUser can be moved here.
	// For brevity, I'll sketch it out; the full detail is in your existing UserHandler.
	dbUser, err := s.userRepo.GetAdmin()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get admin user")
	}
	originalAdminUsername := dbUser.Username

	var conv converter.DtoToDbomConverterImpl
	if err := conv.UserToSambaUser(userDto, &dbUser); err != nil {
		return nil, errors.Wrap(err, "failed to convert admin DTO to DBOM")
	}
	dbUser.IsAdmin = true // Ensure admin status

	if dbUser.Username != originalAdminUsername {
		if _, checkErr := s.userRepo.GetUserByName(dbUser.Username); checkErr == nil && dbUser.Username != originalAdminUsername {
			return nil, errors.WithMessagef(dto.ErrorUserAlreadyExists, "cannot rename admin to %s, username already exists", dbUser.Username)
		} else if !errors.Is(checkErr, gorm.ErrRecordNotFound) && dbUser.Username != originalAdminUsername {
			return nil, errors.Wrapf(checkErr, "error checking if new admin username %s exists", dbUser.Username)
		}
		if err := s.userRepo.Rename(originalAdminUsername, dbUser.Username); err != nil {
			return nil, errors.Wrapf(err, "failed to rename admin user from %s to %s", originalAdminUsername, dbUser.Username)
		}
	}

	if err := s.userRepo.Save(&dbUser); err != nil {
		return nil, errors.Wrap(err, "failed to save updated admin user")
	}

	s.dirtyService.SetDirtyUsers()
	go s.shareService.NotifyClient()

	var updatedAdminDto dto.User
	if err := conv.SambaUserToUser(dbUser, &updatedAdminDto); err != nil {
		return nil, errors.Wrap(err, "failed to convert updated admin DBOM to DTO")
	}
	return &updatedAdminDto, nil
}

func (s *UserService) DeleteUser(username string) error {
	if err := s.userRepo.Delete(username); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.ErrorUserNotFound
		}
		return errors.Wrapf(err, "failed to delete user %s from repository", username)
	}
	s.dirtyService.SetDirtyUsers()
	go s.shareService.NotifyClient()
	return nil
}
