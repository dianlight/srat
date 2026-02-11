package service

import (
	"context"
	"log/slog"
	"os"

	"github.com/angusgmorrison/logfusc"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbom/g"
	"github.com/dianlight/srat/dbom/g/query"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/unixsamba"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

const defaultAdminUsername = "admin"
const defaultAdminPassword = "changeme!"

type UserServiceInterface interface {
	ListUsers() ([]dto.User, error)
	GetAdmin() (*dto.User, error)
	CreateUser(user dto.User) (*dto.User, error)
	UpdateUser(currentUsername string, userDto dto.User) (*dto.User, error)
	UpdateAdminUser(userDto dto.User) (*dto.User, error)
	DeleteUser(username string) error
}

type UserService struct {
	//userRepo       repository.SambaUserRepositoryInterface
	db             *gorm.DB
	ctx            context.Context
	eventBus       events.EventBusInterface
	settingService SettingServiceInterface
}

type UserServiceParams struct {
	fx.In
	Db  *gorm.DB
	Ctx context.Context
	//UserRepo       repository.SambaUserRepositoryInterface
	SettingService SettingServiceInterface
	EventBus       events.EventBusInterface
	//DefaultConfig  *config.DefaultConfig
}

func NewUserService(lc fx.Lifecycle, params UserServiceParams) UserServiceInterface {
	us := &UserService{
		//userRepo:       params.UserRepo,
		ctx:            params.Ctx,
		db:             params.Db,
		eventBus:       params.EventBus,
		settingService: params.SettingService,
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if os.Getenv("SRAT_MOCK") == "true" {
				return nil
			}
			slog.DebugContext(ctx, "******* Autocreating users ********")

			setting, err := us.settingService.Load()
			if err != nil {
				slog.ErrorContext(ctx, "Cant load settings", "err", err)
			}
			HASmbPassword := setting.HASmbPassword.Expose()
			if HASmbPassword == "" {
				slog.ErrorContext(ctx, "Cant get HASmbPassword setting (regenerated password will be used)")
				newPwd, errc := osutil.GenerateSecurePassword()
				if errc != nil {
					slog.ErrorContext(ctx, "Cant generate secure password", "err", errc)
					HASmbPassword = "changeme!"
				} else {
					HASmbPassword = newPwd
				}
				setting.HASmbPassword = logfusc.NewSecret(HASmbPassword)
				us.settingService.UpdateSettings(setting)
			}
			err = unixsamba.CreateSambaUser("_ha_mount_user_", HASmbPassword, unixsamba.UserOptions{
				CreateHome:    false,
				SystemAccount: false,
				Shell:         "/sbin/nologin",
			})
			if err != nil {
				slog.ErrorContext(ctx, "Cant create samba user", "err", err)
			}
			users, errS := gorm.G[dbom.SambaUser](us.db).Find(us.ctx) //.us.userRepo.All()
			if errS != nil {
				slog.ErrorContext(ctx, "Cant load users", "err", errS)
			}
			if len(users) == 0 {
				// Create adminUser
				users = append(users, dbom.SambaUser{
					Username: defaultAdminUsername,
					Password: defaultAdminPassword,
					IsAdmin:  true,
				})
				err := gorm.G[dbom.SambaUser](us.db).Create(us.ctx, &users[0])
				if err != nil {
					slog.ErrorContext(ctx, "Error autocreating admin user", "name", defaultAdminUsername, "err", err)
				}
			}
			for _, user := range users {
				slog.InfoContext(ctx, "Autocreating user", "name", user.Username)
				err = unixsamba.CreateSambaUser(user.Username, user.Password, unixsamba.UserOptions{
					CreateHome:    false,
					SystemAccount: false,
					Shell:         "/sbin/nologin",
				})
				if err != nil {
					slog.ErrorContext(ctx, "Error autocreating user", "name", user.Username, "err", err)
				}
			}
			slog.DebugContext(ctx, "******* Autocreating users done! ********")
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
	dbusers, err := gorm.G[dbom.SambaUser](s.db).Find(s.ctx)
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
	dbuser, err := query.SambaUserQuery[dbom.SambaUser](s.db).GetAdmin(s.ctx)
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

	upd, err := gorm.G[dbom.SambaUser](s.db).
		Scopes(dbom.IncludeSoftDeleted).
		Where(g.SambaUser.Username.Eq(dbUser.Username)).
		Where(g.SambaUser.DeletedAt.IsNotNull()).
		Update(s.ctx, g.SambaUser.DeletedAt.Column().Name, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to restore soft-deleted user %s if existed", dbUser.Username)
	}
	if upd > 0 {
		return s.UpdateUser(userDto.Username, userDto)
	}

	slog.Debug("Attempting to create user in DB", "dbUser", dbUser)
	if err := gorm.G[dbom.SambaUser](s.db).Create(s.ctx, &dbUser); err != nil {
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
	dbUser, err := gorm.G[dbom.SambaUser](s.db).Where(g.SambaUser.Username.Eq(currentUsername)).First(s.ctx)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, dto.ErrorUserNotFound
		}
		return nil, errors.Wrapf(err, "failed to get user %s from repository %v", currentUsername, err)
	}

	var conv converter.DtoToDbomConverterImpl

	if userDto.Username != "" && userDto.Username != currentUsername {
		// Handle rename
		if _, checkErr := gorm.G[dbom.SambaUser](s.db).Where(g.SambaUser.Username.Eq(userDto.Username)).First(s.ctx); checkErr == nil {
			return nil, errors.WithMessagef(dto.ErrorUserAlreadyExists, "cannot rename to %s, user already exists", userDto.Username)
		} else if !errors.Is(checkErr, gorm.ErrRecordNotFound) {
			return nil, errors.Wrapf(checkErr, "error checking if new username %s exists", userDto.Username)
		}
		if err := s.rename(currentUsername, userDto.Username); err != nil {
			return nil, errors.Wrapf(err, "failed to rename user from %s to %s. err %v", currentUsername, userDto.Username, err)
		}
		dbUser.Username = userDto.Username // Update username in the struct for subsequent save
	}

	// Apply other DTO changes to dbUser
	// Ensure the DTO's username is consistent with dbUser.Username for the converter
	originalDtoUsername := userDto.Username
	userDto.Username = dbUser.Username
	if err := conv.UserToSambaUser(userDto, &dbUser); err != nil {
		userDto.Username = originalDtoUsername // Restore for error message consistency
		return nil, errors.Wrap(err, "failed to convert user DTO to DBOM for update")
	}
	userDto.Username = originalDtoUsername // Restore for caller, if they inspect it

	if _, err := gorm.G[dbom.SambaUser](s.db).Updates(s.ctx, dbUser); err != nil {
		return nil, errors.Wrapf(err, "failed to save updated user %s to repository", dbUser.Username)
	}

	var updatedUserDto dto.User
	if err := conv.SambaUserToUser(dbUser, &updatedUserDto); err != nil {
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
	dbUser, err := query.SambaUserQuery[dbom.SambaUser](s.db).GetAdmin(s.ctx)
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
		if _, checkErr := gorm.G[dbom.SambaUser](s.db).Where(g.SambaUser.Username.Eq(dbUser.Username)).First(s.ctx); checkErr == nil {
			return nil, errors.WithMessagef(dto.ErrorUserAlreadyExists, "cannot rename admin from %s to %s, username already exists", originalAdminUsername, dbUser.Username)
		} else if !errors.Is(checkErr, gorm.ErrRecordNotFound) && dbUser.Username != originalAdminUsername {
			return nil, errors.Wrapf(checkErr, "error checking if new admin username %s exists", dbUser.Username)
		}
		if err := s.rename(originalAdminUsername, dbUser.Username); err != nil {
			return nil, errors.Wrapf(err, "failed to rename admin user from %s to %s", originalAdminUsername, dbUser.Username)
		}
	}

	if _, err := gorm.G[dbom.SambaUser](s.db).Updates(s.ctx, dbUser); err != nil {
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
	found, err := query.SambaUserQuery[dbom.SambaUser](s.db).DeleteByName(s.ctx, username)
	if errors.Is(err, gorm.ErrRecordNotFound) || found == 0 {
		return dto.ErrorUserNotFound
	}
	if err != nil {
		return errors.Wrapf(err, "failed to delete user %s from repository %v", username, err)
	}
	s.eventBus.EmitUser(events.UserEvent{
		Event: events.Event{
			Type: events.EventTypes.REMOVE,
		},
		User: &dto.User{Username: username},
	})
	return nil
}

func (self *UserService) rename(oldname string, newname string) errors.E {
	return errors.WithStack(self.db.Transaction(func(tx *gorm.DB) error {
		var smbuser dbom.SambaUser
		// First, retrieve the user to get the password *before* updating the name.
		// We need the original password for the unixsamba.RenameUsername call.
		if err := tx.Where("username = ?", oldname).First(&smbuser).Error; err != nil {
			return errors.Wrapf(err, "failed to find user %s before renaming", oldname)
		}

		if os.Getenv("SRAT_MOCK") != "true" {
			// Attempt to rename the user in the underlying system (Samba/Unix) first
			if err := unixsamba.RenameUsername(oldname, newname, false, smbuser.Password); err != nil {
				return errors.Wrapf(err, "failed to rename user in unix/samba from %s to %s", oldname, newname)
			}
		}

		// Update the username in the database
		if err := tx.Model(&dbom.SambaUser{}).Where("username = ?", oldname).Update("username", newname).Error; err != nil {
			return errors.Wrapf(err, "failed to update username in database from %s to %s", oldname, newname)
		}
		return nil
	}))
}
