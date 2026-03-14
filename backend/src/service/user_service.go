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
	"github.com/dianlight/tlog"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

const defaultAdminUsername = "admin"
const defaultAdminPassword = "changeme!"

// NormalizeUsernameForUnixSamba strips spaces from Samba usernames for Unix account mapping.

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
			tlog.TraceContext(ctx, "******* Autocreating users ********")

			setting, err := us.settingService.Load()
			if err != nil {
				tlog.ErrorContext(ctx, "Cant load settings", "err", err)
			}
			if setting == nil {
				slog.WarnContext(ctx, "Settings are nil, using default startup settings")
				setting = &dto.Settings{}
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
			err = unixsamba.CreateSambaUser(ctx, "_ha_mount_user_", HASmbPassword, unixsamba.UserOptions{
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
				tlog.InfoContext(ctx, "Autocreating user", "name", user.Username)
				err = unixsamba.CreateSambaUser(ctx, user.Username, user.Password, unixsamba.UserOptions{
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
	dbusers, err := gorm.G[dbom.SambaUser](s.db).
		Preload(g.SambaUser.RoShares.Name(), nil).
		Preload(g.SambaUser.RwShares.Name(), nil).
		Find(s.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list users from repository")
	}
	var conv converter.DtoToDbomConverterImpl
	users, err := conv.SambaUsersToUsers(dbusers)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert db users to dto")
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
	var conv converter.DtoToDbomConverterImpl
	user, errS := conv.SambaUserToUser(dbuser)
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
		Select("Password", "DeletedAt"). // Explicitly include DeletedAt in the UPDATE
		Updates(s.ctx, dbom.SambaUser{
			DeletedAt: gorm.DeletedAt{Valid: false}, // Clear DeletedAt to restore
			Password:  dbUser.Password,
		})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to restore soft-deleted user %s if existed", dbUser.Username)
	}
	if upd > 0 {
		// Restore Soft Deleted
		if num, err := gorm.G[dbom.SambaUser](s.db).
			Where(g.SambaUser.Username.Eq(dbUser.Username)).
			Updates(s.ctx, dbUser); err != nil && num == 0 {
			return nil, errors.Wrapf(err, "failed to save updated user %s to repository", dbUser.Username)
		}
	} else if upd == 0 {
		if err := gorm.G[dbom.SambaUser](s.db).Create(s.ctx, &dbUser); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, dto.ErrorUserAlreadyExists
			}
			//tlog.Error("Error creating user in repository", "err", err)
			return nil, errors.Wrap(err, "failed to create user in repository")
		}
	}
	err = unixsamba.CreateSambaUser(s.ctx, dbUser.Username, dbUser.Password, unixsamba.UserOptions{
		CreateHome:    false,
		SystemAccount: false,
		Shell:         "/sbin/nologin",
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to restore samba user %s for soft-deleted user", dbUser.Username)
	}

	//slog.Debug("Attempting to create user in DB", "dbUser", dbUser)

	createdUserDto, err := conv.SambaUserToUser(dbUser)
	if err != nil {
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

	currentPassword := dbUser.Password // Store current password for potential use in rename or password change

	var conv converter.DtoToDbomConverterImpl

	// Apply other DTO changes to dbUser without affecting the username (which is already handled)
	if err := conv.UserToSambaUser(userDto, &dbUser); err != nil {
		return nil, errors.Wrap(err, "failed to convert user DTO to DBOM for update")
	}

	dbUser, err = s.updateUser(currentUsername, currentPassword, dbUser) // Update the user and handle renaming if needed
	if err != nil {
		return nil, err
	}

	updatedUserDto, err := conv.SambaUserToUser(dbUser)
	if err != nil {
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

func (s *UserService) updateUser(currentUsername string, currentPassword string, dbUser dbom.SambaUser) (dbom.SambaUser, error) {

	errF := s.db.Transaction(func(tx *gorm.DB) error {

		if dbUser.Username != "" && dbUser.Username != currentUsername {
			// Handle rename
			if _, checkErr := gorm.G[dbom.SambaUser](tx).Where(g.SambaUser.Username.Eq(dbUser.Username)).First(s.ctx); checkErr == nil {
				return errors.WithMessagef(dto.ErrorUserAlreadyExists, "cannot rename to %s, user already exists", dbUser.Username)
			} else if !errors.Is(checkErr, gorm.ErrRecordNotFound) {
				return errors.Wrapf(checkErr, "error checking if new username %s exists", dbUser.Username)
			}
			if err := unixsamba.RenameUsername(tx.Statement.Context, currentUsername, dbUser.Username, dbUser.Password); err != nil {
				return errors.Wrapf(err, "failed to rename user in unix/samba from %s to %s", currentUsername, dbUser.Username)
			}
			// Rename the primary key in DB as a targeted single-column update so that
			// ON UPDATE CASCADE propagates to the user_rw_share / user_ro_share join tables.
			if err := tx.Model(&dbom.SambaUser{}).Where("username = ?", currentUsername).Update("username", dbUser.Username).Error; err != nil {
				return errors.Wrapf(err, "failed to rename username in DB from %s to %s", currentUsername, dbUser.Username)
			}
			// Advance the local variable so the general Updates below targets the new PK.
			currentUsername = dbUser.Username
		}
		if dbUser.Password != "" && dbUser.Password != currentPassword {
			// Handle change password
			err := unixsamba.ChangePassword(tx.Statement.Context, dbUser.Username, dbUser.Password)
			if err != nil {
				return errors.Wrapf(err, "failed to change password for user %s", dbUser.Username)
			}
		}

		if _, err := gorm.G[dbom.SambaUser](tx).
			Where(g.SambaUser.Username.Eq(currentUsername)).
			Updates(tx.Statement.Context, dbUser); err != nil {
			return errors.Wrapf(err, "failed to save updated user %s to repository", dbUser.Username)
		}
		return nil
	})

	return dbUser, errF
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
	originalAdminPassword := dbUser.Password

	var conv converter.DtoToDbomConverterImpl
	if err := conv.UserToSambaUser(userDto, &dbUser); err != nil {
		return nil, errors.Wrap(err, "failed to convert admin DTO to DBOM")
	}
	dbUser.IsAdmin = true // Ensure admin status

	dbUser, err = s.updateUser(originalAdminUsername, originalAdminPassword, dbUser) // Update the user and handle renaming if needed
	if err != nil {
		return nil, err
	}

	updatedAdminDto, err := conv.SambaUserToUser(dbUser)
	if err != nil {
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

	err = unixsamba.DeleteSambaUser(s.ctx, username)
	if err != nil {
		return errors.Wrapf(err, "failed to delete samba user %s", username)
	}

	s.eventBus.EmitUser(events.UserEvent{
		Event: events.Event{
			Type: events.EventTypes.REMOVE,
		},
		User: &dto.User{Username: username},
	})
	return nil
}
