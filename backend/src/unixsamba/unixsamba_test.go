package unixsamba_test

import (
	"fmt"
	"math/rand/v2"
	"os/user"
	"testing"
	"time"

	"github.com/dianlight/srat/unixsamba" // Adjust if your module path is different
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
)

type UnixSambaTestSuite struct {
	suite.Suite
	mockCmdExec unixsamba.CommandExecutor
	mockOSUser  unixsamba.OSUserLookuper
}

func TestUnixSambaTestSuite(t *testing.T) {
	suite.Run(t, new(UnixSambaTestSuite))
}

func (s *UnixSambaTestSuite) SetupTest() {
	ctrl := mock.NewMockController(s.T())
	//defer ctrl.Finish()

	s.mockCmdExec = mock.Mock[unixsamba.CommandExecutor](ctrl)
	s.mockOSUser = mock.Mock[unixsamba.OSUserLookuper](ctrl)

	unixsamba.SetCommandExecutor(s.mockCmdExec)
	unixsamba.SetOSUserLookuper(s.mockOSUser)
}

func (s *UnixSambaTestSuite) TearDownTest() {
	unixsamba.ResetExecutorsToDefaults()
}

// --- GetByUsername Tests ---

func (s *UnixSambaTestSuite) TestGetByUsername_Success_SambaUserExists() {
	username := "testuser"
	sysUser := &user.User{
		Uid:      "1001",
		Gid:      "1001",
		Username: username,
		Name:     "Test User Gecos",
		HomeDir:  "/home/testuser",
	}
	pdbeditOutput := `
Unix username:        testuser
User SID:             S-1-5-21-xxxx-1001
Primary Group SID:    S-1-5-21-xxxx-513
Password last set:    1679800000
Last logon:           1711447200
	`
	mock.When(s.mockOSUser.Lookup(username)).ThenReturn(sysUser, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))

	info, err := unixsamba.GetByUsername(s.T().Context(), username)

	s.Require().NoError(err)
	s.Require().NotNil(info)
	s.Equal(sysUser.Uid, info.UID)
	s.Equal(sysUser.Gid, info.GID)
	s.Equal(sysUser.Username, info.Username)
	s.Equal(sysUser.Name, info.Name)
	s.Equal(sysUser.HomeDir, info.HomeDir)
	s.True(info.IsSambaUser)
	s.Equal("S-1-5-21-xxxx-1001", info.SambaSID)
	s.Equal("S-1-5-21-xxxx-513", info.SambaPrimaryNT)
	s.True(info.SambaPasswordSet)
	s.Equal(time.Unix(1711447200, 0), info.LastLogon)
}

func (s *UnixSambaTestSuite) TestGetByUsername_Success_SambaUserNotExists() {
	username := "testuser"
	sysUser := &user.User{Uid: "1001", Gid: "1001", Username: username}

	mock.When(s.mockOSUser.Lookup(username)).ThenReturn(sysUser, nil).Verify(matchers.Times(1))
	pdbeditErr := errors.WithDetails(errors.New("pdbedit failed"), "desc", "command execution failed",
		"stderr", "No such user testuser",
	)
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).ThenReturn("", pdbeditErr).Verify(matchers.Times(1))

	info, err := unixsamba.GetByUsername(s.T().Context(), username)

	s.Require().NoError(err)
	s.Require().NotNil(info)
	s.False(info.IsSambaUser)
	s.Empty(info.SambaSID)
}

func (s *UnixSambaTestSuite) TestGetByUsername_SystemUserNotFound() {
	username := "nosuchuser"
	lookupErr := user.UnknownUserError(username)

	mock.When(s.mockOSUser.Lookup(username)).ThenReturn(nil, lookupErr).Verify(matchers.Times(1))

	info, err := unixsamba.GetByUsername(s.T().Context(), username)

	s.Require().Error(err)
	s.Nil(info)
	s.True(errors.Is(err, lookupErr))
}

func (s *UnixSambaTestSuite) TestGetByUsername_PdbeditCommandFails() {
	username := "testuser"
	sysUser := &user.User{Uid: "1001", Gid: "1001", Username: username}
	pdbeditCmdErr := errors.WithDetails(errors.New("some pdbedit error"), "desc", "command execution failed", "stderr", "some other error")

	mock.When(s.mockOSUser.Lookup(username)).ThenReturn(sysUser, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).ThenReturn("", pdbeditCmdErr).Verify(matchers.Times(1))

	info, err := unixsamba.GetByUsername(s.T().Context(), username)

	s.Require().Error(err)
	s.NotNil(info) // Info struct might be partially filled before the error
	s.False(info.IsSambaUser)

	var e errors.E
	s.Require().True(errors.As(err, &e), "Error should be a tozd/go/errors.E")
	s.Contains(e.Error(), "pdbedit check for samba user 'testuser' failed")
}

// --- CreateSambaUser Tests ---

func (s *UnixSambaTestSuite) TestCreateSambaUser_Success_NewUser() {
	username := "newuser"
	password := "password123"
	options := unixsamba.UserOptions{CreateHome: true, Shell: "/bin/bash"}
	// NT hash for "password123" = A9FDFA038C4B75EBC76DC855DD74F0DA
	smbPasswdLine := username + ":1001:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:A9FDFA038C4B75EBC76DC855DD74F0DA:[U          ]:::"
	pdbeditVerbose := "Unix username:        newuser\nAccount Flags:        [U          ]\n"

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "useradd", "-s", "/bin/bash", "--badname", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommandWithInput(s.T().Context(), password+"\n"+password+"\n", "smbpasswd", "-a", "-s", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).ThenReturn(pdbeditVerbose, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-w", "-u", username)).ThenReturn(smbPasswdLine, nil).Verify(matchers.Times(1))

	err := unixsamba.CreateSambaUser(s.T().Context(), username, password, options)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestCreateSambaUser_Success_SystemUserExists() {
	username := "existinguser"
	password := "password123"
	options := unixsamba.UserOptions{}
	// NT hash for "password123" = A9FDFA038C4B75EBC76DC855DD74F0DA
	smbPasswdLine := username + ":1001:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:A9FDFA038C4B75EBC76DC855DD74F0DA:[U          ]:::"
	pdbeditVerbose := "Unix username:        existinguser\nAccount Flags:        [U          ]\n"

	useraddErr := errors.WithDetails(errors.New("useradd failed"), "desc", "command execution failed",
		"stderr", "useradd: user 'existinguser' already exists",
	)
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "useradd", "-M", "--badname", username)).ThenReturn("", useraddErr).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommandWithInput(s.T().Context(), password+"\n"+password+"\n", "smbpasswd", "-a", "-s", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).ThenReturn(pdbeditVerbose, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-w", "-u", username)).ThenReturn(smbPasswdLine, nil).Verify(matchers.Times(1))

	err := unixsamba.CreateSambaUser(s.T().Context(), username, password, options)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestCreateSambaUser_UseraddFails_UserNotExists() {
	username := fmt.Sprintf("newuser5%d", rand.IntN(100))
	password := "password123"
	options := unixsamba.UserOptions{}
	useraddActualErr := errors.New("some useradd error")
	useraddCmdErr := errors.WithDetails(useraddActualErr, "desc", "command execution failed", "stderr", "some useradd error")

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "useradd", "-M", "--badname", username)).ThenReturn("", useraddCmdErr).Verify(matchers.Times(1))
	mock.When(s.mockOSUser.Lookup(username)).ThenReturn(nil, errors.New("user not found")).Verify(matchers.Times(1))

	err := unixsamba.CreateSambaUser(s.T().Context(), username, password, options)
	s.Require().Error(err)
	s.Contains(err.Error(), fmt.Sprintf("failed to create system user '%s'", username))
	s.True(errors.Is(err, useraddCmdErr))
}

func (s *UnixSambaTestSuite) TestCreateSambaUser_SmbPasswdFails() {
	username := "newuser"
	password := "password123"
	options := unixsamba.UserOptions{}
	smbPasswdActualErr := errors.New("smbpasswd error")
	smbPasswdCmdErr := errors.WithDetails(smbPasswdActualErr, "desc", "command execution with input failed", "stderr", "smb error")

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "useradd", "-M", "--badname", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommandWithInput(s.T().Context(), password+"\n"+password+"\n", "smbpasswd", "-a", "-s", username)).ThenReturn("", smbPasswdCmdErr).Verify(matchers.Times(1))

	err := unixsamba.CreateSambaUser(s.T().Context(), username, password, options)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to add user 'newuser' to Samba or set password")
	s.True(errors.Is(err, smbPasswdCmdErr))
}

func (s *UnixSambaTestSuite) TestCreateSambaUser_WithOptions() {
	username := "optionsuser"
	password := "securepass"
	// NT hash for "securepass" = 4B1F924A6A133F726392E60B32E425CA
	smbPasswdLine := username + ":1001:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:4B1F924A6A133F726392E60B32E425CA:[U          ]:::"
	pdbeditVerbose := "Unix username:        optionsuser\nAccount Flags:        [U          ]\n"
	options := unixsamba.UserOptions{
		HomeDir:       "/var/customhome",
		Shell:         "/sbin/nologin",
		PrimaryGroup:  "customgroup",
		GECOS:         []string{"group1", "group2"},
		CreateHome:    false,
		SystemAccount: true,
		UID:           "2001",
	}

	expectedUseraddArgs := []string{
		"-M", "-r", // CreateHome, SystemAccount
		"-d", "/var/customhome",
		"-s", "/sbin/nologin",
		"-G", "customgroup",
		"-g", "group1,group2",
		"-u", "2001",
		"--badname",
		username,
	}

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "useradd",
		expectedUseraddArgs[0],
		expectedUseraddArgs[1], expectedUseraddArgs[2],
		expectedUseraddArgs[3], expectedUseraddArgs[4],
		expectedUseraddArgs[5], expectedUseraddArgs[6],
		expectedUseraddArgs[7], expectedUseraddArgs[8],
		expectedUseraddArgs[9], expectedUseraddArgs[10],
		expectedUseraddArgs[11], expectedUseraddArgs[12],
		expectedUseraddArgs[13],
	)).
		ThenReturn("", nil).
		Verify(matchers.Times(1))

	mock.When(s.mockCmdExec.RunCommandWithInput(s.T().Context(), password+"\n"+password+"\n", "smbpasswd", "-a", "-s", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).ThenReturn(pdbeditVerbose, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-w", "-u", username)).ThenReturn(smbPasswdLine, nil).Verify(matchers.Times(1))

	err := unixsamba.CreateSambaUser(s.T().Context(), username, password, options)
	s.Require().NoError(err)
}

// --- DeleteSambaUser Tests ---

func (s *UnixSambaTestSuite) TestDeleteSambaUser_Success() {
	username := "sysdeluser"
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "smbpasswd", "-x", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "deluser", "--remove-home", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).ThenReturn("", nil).Verify(matchers.Times(1))

	err := unixsamba.DeleteSambaUser(s.T().Context(), username)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestDeleteSambaUser_SmbPasswdFails_NotUserNotFound() {
	username := "smbdeluser"
	smbErrActual := errors.New("smb error")
	smbCmdErr := errors.WithDetails(smbErrActual, "desc", "command execution failed", "stderr", "some other smb error")

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "smbpasswd", "-x", username)).ThenReturn("", smbCmdErr).Verify(matchers.Times(1))

	err := unixsamba.DeleteSambaUser(s.T().Context(), username)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to delete user 'smbdeluser' from Samba")
	s.True(errors.Is(err, smbCmdErr))
}

func (s *UnixSambaTestSuite) TestDeleteSambaUser_SmbPasswdUserNotFound_SystemDeleteSuccess() {
	username := "smbnotfound"
	smbCmdErr := errors.WithDetails(errors.New("smb not found"), "desc", "command execution failed",
		"stderr", "Failed to find entry for user smbnotfound.",
	)

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "smbpasswd", "-x", username)).ThenReturn("", smbCmdErr).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "deluser", "--remove-home", username)).ThenReturn("", nil).Verify(matchers.Times(1))

	err := unixsamba.DeleteSambaUser(s.T().Context(), username)
	s.Require().NoError(err) // Error from smbpasswd -x (user not found) is ignored if system deletion is requested and succeeds
}

func (s *UnixSambaTestSuite) TestDeleteSambaUser_UserdelFails() {
	username := "sysdeluser"
	userdelActualErr := errors.New("userdel error")
	userdelCmdErr := errors.WithDetails(userdelActualErr, "desc", "command execution failed", "stderr", "userdel critical error")

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "smbpasswd", "-x", username)).ThenReturn("", nil).Verify(matchers.Times(1)) // Samba deletion succeeds
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "deluser", "--remove-home", username)).ThenReturn("", userdelCmdErr).Verify(matchers.Times(1))

	err := unixsamba.DeleteSambaUser(s.T().Context(), username)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to delete system user 'sysdeluser'")
	s.True(errors.Is(err, userdelCmdErr))
}

// --- ChangePassword Tests ---

func (s *UnixSambaTestSuite) TestChangePassword_SambaOnly_Success() {
	username := "changepwuser"
	newPassword := "newSecurePassword"
	input := newPassword + "\n" + newPassword + "\n"
	// NT hash for "newSecurePassword" = 7F43E2A648F47B9AE704C3A00ABB6A2D
	smbPasswdLine := username + ":1001:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:7F43E2A648F47B9AE704C3A00ABB6A2D:[U          ]:::"
	pdbeditVerbose := "Unix username:        changepwuser\nAccount Flags:        [U          ]\n"

	mock.When(s.mockCmdExec.RunCommandWithInput(s.T().Context(), input, "smbpasswd", "-s", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).ThenReturn(pdbeditVerbose, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-w", "-u", username)).ThenReturn(smbPasswdLine, nil).Verify(matchers.Times(1))

	err := unixsamba.ChangePassword(s.T().Context(), username, newPassword)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestChangePassword_SambaAndSystem_Success() {
	username := "changepwuser"
	newPassword := "newSecurePassword"
	sambaInput := newPassword + "\n" + newPassword + "\n"
	//chpasswdInput := username + ":" + newPassword + "\n"
	// NT hash for "newSecurePassword" = 7F43E2A648F47B9AE704C3A00ABB6A2D
	smbPasswdLine := username + ":1001:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:7F43E2A648F47B9AE704C3A00ABB6A2D:[U          ]:::"
	pdbeditVerbose := "Unix username:        changepwuser\nAccount Flags:        [U          ]\n"

	mock.When(s.mockCmdExec.RunCommandWithInput(s.T().Context(), sambaInput, "smbpasswd", "-s", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	//mock.When(s.mockCmdExec.RunCommandWithInput(s.T().Context(), chpasswdInput, "chpasswd")).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).ThenReturn(pdbeditVerbose, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-w", "-u", username)).ThenReturn(smbPasswdLine, nil).Verify(matchers.Times(1))

	err := unixsamba.ChangePassword(s.T().Context(), username, newPassword)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestChangePassword_SmbPasswdFails() {
	username := "changepwuser"
	newPassword := "newSecurePassword"
	input := newPassword + "\n" + newPassword + "\n"
	smbErrActual := errors.New("smb error")
	smbCmdErr := errors.WithDetails(smbErrActual, "desc", "command execution with input failed", "stderr", "smb change error")

	mock.When(s.mockCmdExec.RunCommandWithInput(s.T().Context(), input, "smbpasswd", "-s", username)).ThenReturn("", smbCmdErr).Verify(matchers.Times(1))

	err := unixsamba.ChangePassword(s.T().Context(), username, newPassword)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to change Samba password for user 'changepwuser'")
	s.True(errors.Is(err, smbCmdErr))
}

// --- RenameUsername Tests ---

func (s *UnixSambaTestSuite) TestRenameUsername_Success_NoHomeRename() {
	oldUsername := "oldname"
	newUsername := "newname"
	newPassword := "newpass"
	sambaInput := newPassword + "\n" + newPassword + "\n"

	mock.When(s.mockOSUser.Lookup(newUsername)).
		ThenReturn(nil, user.UnknownUserError(newUsername)).                                 // Initial system check
		ThenReturn(&user.User{Username: newUsername, HomeDir: "/home/" + newUsername}, nil). // Post-rename home check
		Verify(matchers.Times(2))
	pdbeditNotFoundErr := errors.WithDetails(errors.New("pdbedit failed"), "desc", "command execution failed", "stderr", "No such user")
	pdbeditVerbose := "Unix username:        newname\nAccount Flags:        [U          ]\n"
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", newUsername)).
		ThenReturn("", pdbeditNotFoundErr). // Pre-rename samba existence check
		ThenReturn(pdbeditVerbose, nil).    // CheckSambaUser verification
		Verify(matchers.Times(2))

	// 3. usermod -l
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "usermod", "-l", newUsername, oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1))

	// 4. smbpasswd -x oldname
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "smbpasswd", "-x", oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1))

	// 5. smbpasswd -a -s newname
	mock.When(s.mockCmdExec.RunCommandWithInput(s.T().Context(), sambaInput, "smbpasswd", "-a", "-s", newUsername)).ThenReturn("", nil).Verify(matchers.Times(1))

	// 6. CheckSambaUser verification: pdbedit -L -v and pdbedit -L -w for newUsername
	// NT hash for "newpass" = 18DA6C2895C549E266745951D5DC66CB
	smbPasswdLine := newUsername + ":1001:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:18DA6C2895C549E266745951D5DC66CB:[U          ]:::"
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-w", "-u", newUsername)).ThenReturn(smbPasswdLine, nil).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, newPassword)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestRenameUsername_Success_WithHomeRename() {
	oldUsername := "oldhome"
	newUsername := "newhome"
	newPassword := "newpass"
	sambaInput := newPassword + "\n" + newPassword + "\n"
	expectedNewHomeDir := "/home/" + newUsername

	mock.When(s.mockOSUser.Lookup(newUsername)).
		ThenReturn(nil, user.UnknownUserError(newUsername)).                                 // Initial check
		ThenReturn(&user.User{Username: newUsername, HomeDir: "/home/" + oldUsername}, nil). // Post-rename home check
		Verify(matchers.Times(2))
	pdbeditErr := errors.WithDetails(errors.New("pdbedit failed"), "desc", "command execution failed", "stderr", "No such user")
	pdbeditVerbose := "Unix username:        newhome\nAccount Flags:        [U          ]\n"
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", newUsername)).
		ThenReturn("", pdbeditErr).      // Pre-rename samba existence check
		ThenReturn(pdbeditVerbose, nil). // CheckSambaUser verification
		Verify(matchers.Times(2))

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "usermod", "-l", newUsername, oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1))

	// For home dir rename part
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "usermod", "-d", expectedNewHomeDir, "-m", newUsername)).ThenReturn("", nil).Verify(matchers.Times(1))

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "smbpasswd", "-x", oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommandWithInput(s.T().Context(), sambaInput, "smbpasswd", "-a", "-s", newUsername)).ThenReturn("", nil).Verify(matchers.Times(1))

	// CheckSambaUser verification: pdbedit -L -v and pdbedit -L -w for newUsername
	// NT hash for "newpass" = 18DA6C2895C549E266745951D5DC66CB
	smbPasswdLine := newUsername + ":1001:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:18DA6C2895C549E266745951D5DC66CB:[U          ]:::"
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-w", "-u", newUsername)).ThenReturn(smbPasswdLine, nil).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, newPassword)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_NewUserSystemExists() {
	oldUsername := "old"
	newUsername := "existing"
	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(&user.User{Username: newUsername}, nil).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, "pass")
	s.Require().Error(err)
	s.EqualError(err, "new username 'existing' already exists on the system")
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_NewUserSambaExists() {
	oldUsername := "old"
	newUsername := "sambanew"

	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(nil, user.UnknownUserError(newUsername)).Verify(matchers.Times(1))

	// pdbedit finds a samba user.
	pdbeditOutput := "User SID: S-1-5-blah"
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", newUsername)).ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, "pass")
	s.Require().Error(err)
	s.EqualError(err, "new username 'sambanew' already appears to be a Samba user", "err", errors.Unwrap(err))
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_PdbeditIssueForNewUser() {

	oldUsername := "old"
	newUsername := "pdbissue"

	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(nil, user.UnknownUserError(newUsername)).Verify(matchers.Times(1))

	// pdbedit check for newUsername encounters execution error
	pdbeditActualErr := errors.New("pdbedit command failed")
	pdbeditCmdErr := errors.WithDetails(pdbeditActualErr, "desc", "command execution failed", "command", "pdbedit", "stderr", "critical pdbedit error")
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", newUsername)).ThenReturn("", pdbeditCmdErr).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, "pass")
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to verify Samba status for new username 'pdbissue' due to pdbedit execution issue")
	s.True(errors.Is(err, pdbeditCmdErr))
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_UsermodLoginFails() {

	oldUsername := "old"
	newUsername := "new"
	usermodErrActual := errors.New("usermod error")
	usermodCmdErr := errors.WithDetails(usermodErrActual, "desc", "command execution failed", "stderr", "usermod fail")

	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(nil, user.UnknownUserError(newUsername)).Verify(matchers.Times(1))
	pdbeditErr := errors.WithDetails(errors.New("pdbedit failed"), "desc", "command execution failed", "stderr", "No such user")
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", newUsername)).ThenReturn("", pdbeditErr).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "smbpasswd", "-x", oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "usermod", "-l", newUsername, oldUsername)).ThenReturn("", usermodCmdErr).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, "pass")
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to rename system user login")
	s.True(errors.Is(err, usermodCmdErr))
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_UsermodHomeFails() {
	oldUsername := "oldhome"
	newUsername := "newhome"
	newPassword := "newpass"
	expectedNewHomeDir := "/home/" + newUsername
	usermodHomeErrActual := errors.New("usermod home error")
	usermodHomeCmdErr := errors.WithDetails(usermodHomeErrActual, "desc", "command execution failed", "stderr", "usermod home fail")

	mock.When(s.mockOSUser.Lookup(newUsername)).
		ThenReturn(nil, user.UnknownUserError(newUsername)).
		ThenReturn(&user.User{Username: newUsername, HomeDir: "/home/" + oldUsername}, nil).
		Verify(matchers.Times(2))
	pdbeditErr := errors.WithDetails(errors.New("pdbedit failed"), "desc", "command execution failed", "stderr", "No such user")
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", newUsername)).ThenReturn("", pdbeditErr).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "smbpasswd", "-x", oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "usermod", "-l", newUsername, oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1))

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "usermod", "-d", expectedNewHomeDir, "-m", newUsername)).ThenReturn("", usermodHomeCmdErr).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, newPassword)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to move/rename home directory")
	s.True(errors.Is(err, usermodHomeCmdErr))
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_NoPasswordForSamba() {
	err := unixsamba.RenameUsername(s.T().Context(), "old", "new", "")
	s.Require().Error(err)
	s.EqualError(err, "a new password must be provided to re-add user to Samba after renaming")
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_SmbPasswdAddFails() {
	oldUsername := "old"
	newUsername := "new"
	newPassword := "newpass"
	sambaInput := newPassword + "\n" + newPassword + "\n"
	smbAddErrActual := errors.New("smb add error")
	smbAddCmdErr := errors.WithDetails(smbAddErrActual, "desc", "command execution with input failed", "stderr", "smb add fail")

	mock.When(s.mockOSUser.Lookup(newUsername)).
		ThenReturn(nil, user.UnknownUserError(newUsername)).
		ThenReturn(&user.User{Username: newUsername, HomeDir: "/home/" + newUsername}, nil).
		Verify(matchers.Times(2))
	pdbeditErr := errors.WithDetails(errors.New("pdbedit failed"), "desc", "command execution failed", "stderr", "No such user")
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", newUsername)).ThenReturn("", pdbeditErr).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "usermod", "-l", newUsername, oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "smbpasswd", "-x", oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1)) // Deletion of old samba user
	mock.When(s.mockCmdExec.RunCommandWithInput(s.T().Context(), sambaInput, "smbpasswd", "-a", "-s", newUsername)).ThenReturn("", smbAddCmdErr).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, newPassword)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to add new Samba user 'new' after renaming")
	s.True(errors.Is(err, smbAddCmdErr))
}

// --- ListSambaUsers Tests ---

func (s *UnixSambaTestSuite) TestListSambaUsers_Success() {
	pdbeditOutput := `
user1:1001:User One
user2:1002:User Two
adminuser:1000:Admin
	`
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L")).ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))

	users, err := unixsamba.ListSambaUsers(s.T().Context())
	s.Require().NoError(err)
	s.Require().ElementsMatch([]string{"user1", "user2", "adminuser"}, users)
}

func (s *UnixSambaTestSuite) TestListSambaUsers_Success_Empty() {
	pdbeditOutput := ""
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L")).ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))

	users, err := unixsamba.ListSambaUsers(s.T().Context())
	s.Require().NoError(err)
	s.Empty(users)
}

func (s *UnixSambaTestSuite) TestListSambaUsers_PdbeditFails() {
	pdbeditActualErr := errors.New("pdbedit list error")
	pdbeditCmdErr := errors.WithDetails(pdbeditActualErr, "desc", "command execution failed", "stderr", "pdbedit -L failed")

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L")).ThenReturn("", pdbeditCmdErr).Verify(matchers.Times(1))

	users, err := unixsamba.ListSambaUsers(s.T().Context())
	s.Require().Error(err)
	s.Nil(users)
	s.Contains(err.Error(), "failed to list samba users with pdbedit -L")
	s.True(errors.Is(err, pdbeditCmdErr))
}

// --- CheckSambaUser Tests ---

func (s *UnixSambaTestSuite) TestCheckSambaUser_Success() {
	username := "activeuser"
	password := "correctpass"
	// smbpasswd format: username:uid:LMHASH:NTHASH:flags:::
	// NT hash for "correctpass" = B2C60158793D1414461032ADD2B99D8B
	smbPasswdLine := username + ":1001:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:B2C60158793D1414461032ADD2B99D8B:[U          ]:::"
	pdbeditOutput := "Unix username:        activeuser\nAccount Flags:        [U          ]\n"

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).
		ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-w", "-u", username)).
		ThenReturn(smbPasswdLine, nil).Verify(matchers.Times(1))

	s.Require().NoError(unixsamba.CheckSambaUser(s.T().Context(), username, password))
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_UserNotFound() {
	username := "nosuchsmbuser"
	password := "somepass"
	pdbeditErr := errors.WithDetails(errors.New("pdbedit failed"), "desc", "command execution failed",
		"stderr", "Username not found: nosuchsmbuser",
	)

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).
		ThenReturn("", pdbeditErr).Verify(matchers.Times(1))

	err := unixsamba.CheckSambaUser(s.T().Context(), username, password)
	s.Require().Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_PdbeditCommandFails() {
	username := "testuser"
	password := "somepass"
	pdbeditErr := errors.WithDetails(errors.New("pdbedit error"), "desc", "command execution failed",
		"stderr", "unexpected pdbedit error",
	)

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).
		ThenReturn("", pdbeditErr).Verify(matchers.Times(1))

	err := unixsamba.CheckSambaUser(s.T().Context(), username, password)
	s.Require().Error(err)
	s.Contains(err.Error(), "pdbedit check for samba user")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_AccountNotActive() {
	username := "nouflag"
	password := "somepass"
	// Account Flags do not contain [U]
	pdbeditOutput := "Unix username:        nouflag\nAccount Flags:        [           ]\n"

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).
		ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))

	err := unixsamba.CheckSambaUser(s.T().Context(), username, password)
	s.Require().Error(err)
	s.Contains(err.Error(), "not active")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_AccountDisabled() {
	username := "disableduser"
	password := "somepass"
	// Account Flags contain both [U] and [D]
	pdbeditOutput := "Unix username:        disableduser\nAccount Flags:        [DU         ]\n"

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).
		ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))

	err := unixsamba.CheckSambaUser(s.T().Context(), username, password)
	s.Require().Error(err)
	s.Contains(err.Error(), "disabled or locked")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_AccountLocked() {
	username := "lockeduser"
	password := "somepass"
	// Account Flags contain [U] and [L]
	pdbeditOutput := "Unix username:        lockeduser\nAccount Flags:        [LU         ]\n"

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).
		ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))

	err := unixsamba.CheckSambaUser(s.T().Context(), username, password)
	s.Require().Error(err)
	s.Contains(err.Error(), "disabled or locked")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_WrongPassword_SmbLogonFailure() {
	username := "testuser"
	password := "wrongpass"
	// Hash stored is for a DIFFERENT password, so comparison must fail.
	smbPasswdLine := username + ":1001:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:B2C60158793D1414461032ADD2B99D8B:[U          ]:::"
	pdbeditOutput := "Unix username:        testuser\nAccount Flags:        [U          ]\n"

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).
		ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-w", "-u", username)).
		ThenReturn(smbPasswdLine, nil).Verify(matchers.Times(1))

	err := unixsamba.CheckSambaUser(s.T().Context(), username, password)
	s.Require().Error(err)
	s.Contains(err.Error(), "invalid password")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_WrongPassword_SmbWrongPassword() {
	username := "testuser"
	password := "wrongpass"
	// Same scenario as SmbLogonFailure — hash mismatch catches wrong passwords.
	smbPasswdLine := username + ":1001:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:B2C60158793D1414461032ADD2B99D8B:[U          ]:::"
	pdbeditOutput := "Unix username:        testuser\nAccount Flags:        [U          ]\n"

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).
		ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-w", "-u", username)).
		ThenReturn(smbPasswdLine, nil).Verify(matchers.Times(1))

	err := unixsamba.CheckSambaUser(s.T().Context(), username, password)
	s.Require().Error(err)
	s.Contains(err.Error(), "invalid password")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_PdbeditHashFails() {
	username := "testuser"
	password := "somepass"
	pdbeditOutput := "Unix username:        testuser\nAccount Flags:        [U          ]\n"
	pdbeditHashErr := errors.WithDetails(errors.New("pdbedit -w error"), "desc", "command execution failed",
		"stderr", "failed to open database",
	)

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).
		ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-w", "-u", username)).
		ThenReturn("", pdbeditHashErr).Verify(matchers.Times(1))

	err := unixsamba.CheckSambaUser(s.T().Context(), username, password)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to read samba password hash")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_NTHashDisabled() {
	username := "testuser"
	password := "somepass"
	// All-X NT hash means no password is set in Samba.
	smbPasswdLine := username + ":1001:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:[U          ]:::"
	pdbeditOutput := "Unix username:        testuser\nAccount Flags:        [U          ]\n"

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).
		ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-w", "-u", username)).
		ThenReturn(smbPasswdLine, nil).Verify(matchers.Times(1))

	err := unixsamba.CheckSambaUser(s.T().Context(), username, password)
	s.Require().Error(err)
	s.Contains(err.Error(), "no password set")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_NTHashParseError() {
	username := "testuser"
	password := "somepass"
	// Return output with no recognisable smbpasswd line for the user.
	pdbeditOutput := "Unix username:        testuser\nAccount Flags:        [U          ]\n"

	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-v", "-u", username)).
		ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand(s.T().Context(), "pdbedit", "-L", "-w", "-u", username)).
		ThenReturn("unrecognised output line", nil).Verify(matchers.Times(1))

	err := unixsamba.CheckSambaUser(s.T().Context(), username, password)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to parse samba password hash")
}
