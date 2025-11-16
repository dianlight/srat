package service

import (
	"context"
	"log/slog"
	"os"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/unixsamba"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type UserServiceInterface interface {
	ListUsers() ([]dto.User, error)
	GetAdmin() (*dto.User, error)
	CreateUser(user dto.User) (*dto.User, error)
	UpdateUser(currentUsername string, userDto dto.User) (*dto.User, error)
	UpdateAdminUser(userDto dto.User) (*dto.User, error)
	DeleteUser(username string) error
}

type UserService struct {
	userRepo       repository.SambaUserRepositoryInterface
	eventBus       events.EventBusInterface
	settingService SettingServiceInterface
}

type UserServiceParams struct {
	fx.In
	UserRepo       repository.SambaUserRepositoryInterface
	SettingService SettingServiceInterface
	EventBus       events.EventBusInterface
	DefaultConfig  *config.DefaultConfig
}

func NewUserService(lc fx.Lifecycle, params UserServiceParams) UserServiceInterface {
	us := &UserService{
		userRepo:       params.UserRepo,
		eventBus:       params.EventBus,
		settingService: params.SettingService,
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if os.Getenv("SRAT_MOCK") == "true" {
				return nil
			}
			slog.Info("******* Autocreating users ********")

			_ha_mount_user_password_, err := us.settingService.GetValue("_ha_mount_user_password_")
			if err != nil {
				slog.Error("Cant get _ha_mount_user_password_ setting", "err", err)
				_ha_mount_user_password_ = "changeme!"
			} else if _ha_mount_user_password_ == nil || _ha_mount_user_password_ == "" {
				_ha_mount_user_password_ = "changeme!"
			}
			err = unixsamba.CreateSambaUser("_ha_mount_user_", _ha_mount_user_password_.(string), unixsamba.UserOptions{
				CreateHome:    false,
				SystemAccount: false,
				Shell:         "/sbin/nologin",
			})
			if err != nil {
				slog.Error("Cant create samba user", "err", err)
			}
			users, err := us.userRepo.All()
			if err != nil {
				slog.Error("Cant load users", "err", err)
			}
			if len(users) == 0 {
				// Create adminUser
				users = append(users, dbom.SambaUser{
					Username: params.DefaultConfig.Username,
					Password: params.DefaultConfig.Password,
					IsAdmin:  true,
				})
				err := us.userRepo.Create(&users[0])
				if err != nil {
					slog.Error("Error autocreating admin user", "name", params.DefaultConfig.Username, "err", err)
				}
			}
			for _, user := range users {
				slog.Info("Autocreating user", "name", user.Username)
				err = unixsamba.CreateSambaUser(user.Username, user.Password, unixsamba.UserOptions{
					CreateHome:    false,
					SystemAccount: false,
					Shell:         "/sbin/nologin",
				})
				if err != nil {
					slog.Error("Error autocreating user", "name", user.Username, "err", err)
				}
			}
			slog.Info("******* Autocreating users done! ********")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Add any cleanup logic here if needed
			return nil
		},
	})
	return us
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
		errS := conv.SambaUserToUser(dbuser, &user)
		if errS != nil {
			return nil, errors.Wrapf(errS, "failed to convert db user %s to dto", dbuser.Username)
		}
		users = append(users, user)
	}
	return users, nil
}

func (s *UserService) GetAdmin() (*dto.User, error) {
	dbuser, err := s.userRepo.GetAdmin()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, dto.ErrorUserNotFound
		}
		return nil, errors.Wrap(err, "failed to get admin user from repository")
	}
	var user dto.User
	var conv converter.DtoToDbomConverterImpl
	errS := conv.SambaUserToUser(dbuser, &user)
	if errS != nil {
		return nil, errors.Wrap(errS, "failed to convert admin db user to dto")
	}
	return &user, nil
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
		//slog.Error("Error creating user in repository", "err", err)
		return nil, errors.Wrap(err, "failed to create user in repository")
	}

	var createdUserDto dto.User
	if err := conv.SambaUserToUser(dbUser, &createdUserDto); err != nil {
		return nil, errors.Wrap(err, "failed to convert created DBOM user back to DTO")
	}

	s.eventBus.EmitUser(events.UserEvent{
		Event: events.Event{
			Type: events.EventTypes.ADD,
		},
		User: &createdUserDto,
	})

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

	var updatedUserDto dto.User
	if err := conv.SambaUserToUser(*dbUser, &updatedUserDto); err != nil {
		return nil, errors.Wrap(err, "failed to convert updated DBOM user back to DTO")
	}

	s.eventBus.EmitUser(events.UserEvent{
		Event: events.Event{
			Type: events.EventTypes.UPDATE,
		},
		User: &updatedUserDto,
	})

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

	var updatedAdminDto dto.User
	if err := conv.SambaUserToUser(dbUser, &updatedAdminDto); err != nil {
		return nil, errors.Wrap(err, "failed to convert updated admin DBOM to DTO")
	}
	s.eventBus.EmitUser(events.UserEvent{
		Event: events.Event{
			Type: events.EventTypes.UPDATE,
		},
		User: &updatedAdminDto,
	})
	return &updatedAdminDto, nil
}

func (s *UserService) DeleteUser(username string) error {
	if err := s.userRepo.Delete(username); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.ErrorUserNotFound
		}
		return errors.Wrapf(err, "failed to delete user %s from repository", username)
	}
	s.eventBus.EmitUser(events.UserEvent{
		Event: events.Event{
			Type: events.EventTypes.REMOVE,
		},
		User: &dto.User{Username: username},
	})
	return nil
}
