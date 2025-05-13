package unixsamba_test

import (
	"fmt"
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
	mock.When(s.mockCmdExec.RunCommand("pdbedit", "-L", "-v", "-u", username)).ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))

	info, err := unixsamba.GetByUsername(username)

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
	mock.When(s.mockCmdExec.RunCommand("pdbedit", "-L", "-v", "-u", username)).ThenReturn("", pdbeditErr).Verify(matchers.Times(1))

	info, err := unixsamba.GetByUsername(username)

	s.Require().NoError(err)
	s.Require().NotNil(info)
	s.False(info.IsSambaUser)
	s.Empty(info.SambaSID)
}

func (s *UnixSambaTestSuite) TestGetByUsername_SystemUserNotFound() {
	username := "nosuchuser"
	lookupErr := user.UnknownUserError(username)

	mock.When(s.mockOSUser.Lookup(username)).ThenReturn(nil, lookupErr).Verify(matchers.Times(1))

	info, err := unixsamba.GetByUsername(username)

	s.Require().Error(err)
	s.Nil(info)
	s.True(errors.Is(err, lookupErr))
}

func (s *UnixSambaTestSuite) TestGetByUsername_PdbeditCommandFails() {
	username := "testuser"
	sysUser := &user.User{Uid: "1001", Gid: "1001", Username: username}
	pdbeditCmdErr := errors.WithDetails(errors.New("some pdbedit error"), "desc", "command execution failed", "stderr", "some other error")

	mock.When(s.mockOSUser.Lookup(username)).ThenReturn(sysUser, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand("pdbedit", "-L", "-v", "-u", username)).ThenReturn("", pdbeditCmdErr).Verify(matchers.Times(1))

	info, err := unixsamba.GetByUsername(username)

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

	mock.When(s.mockCmdExec.RunCommand("useradd", "-m", "-s", "/bin/bash", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommandWithInput(password+"\n"+password+"\n", "smbpasswd", "-a", "-s", username)).ThenReturn("", nil).Verify(matchers.Times(1))

	err := unixsamba.CreateSambaUser(username, password, options)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestCreateSambaUser_Success_SystemUserExists() {
	username := "existinguser"
	password := "password123"
	options := unixsamba.UserOptions{}

	useraddErr := errors.WithDetails(errors.New("useradd failed"), "desc", "command execution failed",
		"stderr", "useradd: user 'existinguser' already exists",
	)
	mock.When(s.mockCmdExec.RunCommand("useradd", username)).ThenReturn("", useraddErr).Verify(matchers.Times(1))
	// Fallback check
	//mock.When(s.mockOSUser.Lookup(username)).ThenReturn(&user.User{Username: username}, nil).Verify(matchers.Times(1))

	mock.When(s.mockCmdExec.RunCommandWithInput(password+"\n"+password+"\n", "smbpasswd", "-a", "-s", username)).ThenReturn("", nil).Verify(matchers.Times(1))

	err := unixsamba.CreateSambaUser(username, password, options)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestCreateSambaUser_UseraddFails_UserNotExists() {
	username := "newuser"
	password := "password123"
	options := unixsamba.UserOptions{}
	useraddActualErr := errors.New("some useradd error")
	useraddCmdErr := errors.WithDetails(useraddActualErr, "desc", "command execution failed", "stderr", "some useradd error")

	mock.When(s.mockCmdExec.RunCommand("useradd", username)).ThenReturn("", useraddCmdErr).Verify(matchers.Times(1))
	// Fallback check, user does not exist
	mock.When(s.mockOSUser.Lookup(username)).ThenReturn(nil, user.UnknownUserError(username)).Verify(matchers.Times(1))

	err := unixsamba.CreateSambaUser(username, password, options)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to create system user 'newuser'")
	s.True(errors.Is(err, useraddCmdErr))
}

func (s *UnixSambaTestSuite) TestCreateSambaUser_SmbPasswdFails() {
	username := "newuser"
	password := "password123"
	options := unixsamba.UserOptions{}
	smbPasswdActualErr := errors.New("smbpasswd error")
	smbPasswdCmdErr := errors.WithDetails(smbPasswdActualErr, "desc", "command execution with input failed", "stderr", "smb error")

	mock.When(s.mockCmdExec.RunCommand("useradd", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommandWithInput(password+"\n"+password+"\n", "smbpasswd", "-a", "-s", username)).ThenReturn("", smbPasswdCmdErr).Verify(matchers.Times(1))

	err := unixsamba.CreateSambaUser(username, password, options)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to add user 'newuser' to Samba or set password")
	s.True(errors.Is(err, smbPasswdCmdErr))
}

func (s *UnixSambaTestSuite) TestCreateSambaUser_WithOptions() {
	username := "optionsuser"
	password := "securepass"
	options := unixsamba.UserOptions{
		HomeDir:         "/var/customhome",
		Shell:           "/sbin/nologin",
		PrimaryGroup:    "customgroup",
		SecondaryGroups: []string{"group1", "group2"},
		CreateHome:      true,
		SystemAccount:   true,
		Comment:         "A test user with options",
		UID:             "2001",
	}

	expectedUseraddArgs := []string{
		"-m", "-r", // CreateHome, SystemAccount
		"-d", "/var/customhome",
		"-s", "/sbin/nologin",
		"-g", "customgroup",
		"-G", "group1,group2",
		"-c", "A test user with options",
		"-u", "2001",
		username,
	}

	mock.When(s.mockCmdExec.RunCommand("useradd",
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

	mock.When(s.mockCmdExec.RunCommandWithInput(password+"\n"+password+"\n", "smbpasswd", "-a", "-s", username)).
		ThenReturn("", nil).
		Verify(matchers.Times(1))

	err := unixsamba.CreateSambaUser(username, password, options)
	s.Require().NoError(err)
}

// --- DeleteSambaUser Tests ---

func (s *UnixSambaTestSuite) TestDeleteSambaUser_SambaOnly_Success() {
	username := "smbdeluser"
	mock.When(s.mockCmdExec.RunCommand("smbpasswd", "-x", username)).ThenReturn("", nil).Verify(matchers.Times(1))

	err := unixsamba.DeleteSambaUser(username, false, false)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestDeleteSambaUser_SambaAndSystem_Success() {
	username := "sysdeluser"
	mock.When(s.mockCmdExec.RunCommand("smbpasswd", "-x", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand("userdel", username)).ThenReturn("", nil).Verify(matchers.Times(1))

	err := unixsamba.DeleteSambaUser(username, true, false)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestDeleteSambaUser_SambaAndSystemWithHome_Success() {
	username := "homedeluser"
	mock.When(s.mockCmdExec.RunCommand("smbpasswd", "-x", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand("userdel", "-r", username)).ThenReturn("", nil).Verify(matchers.Times(1))

	err := unixsamba.DeleteSambaUser(username, true, true)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestDeleteSambaUser_SmbPasswdFails_NotUserNotFound() {
	username := "smbdeluser"
	smbErrActual := errors.New("smb error")
	smbCmdErr := errors.WithDetails(smbErrActual, "desc", "command execution failed", "stderr", "some other smb error")

	mock.When(s.mockCmdExec.RunCommand("smbpasswd", "-x", username)).ThenReturn("", smbCmdErr).Verify(matchers.Times(1))

	err := unixsamba.DeleteSambaUser(username, false, false) // Samba only
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to delete user 'smbdeluser' from Samba")
	s.True(errors.Is(err, smbCmdErr))
}

func (s *UnixSambaTestSuite) TestDeleteSambaUser_SmbPasswdUserNotFound_SystemDeleteSuccess() {
	username := "smbnotfound"
	smbCmdErr := errors.WithDetails(errors.New("smb not found"), "desc", "command execution failed",
		"stderr", "Failed to find entry for user smbnotfound.",
	)

	mock.When(s.mockCmdExec.RunCommand("smbpasswd", "-x", username)).ThenReturn("", smbCmdErr).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand("userdel", username)).ThenReturn("", nil).Verify(matchers.Times(1))

	err := unixsamba.DeleteSambaUser(username, true, false)
	s.Require().NoError(err) // Error from smbpasswd -x (user not found) is ignored if system deletion is requested and succeeds
}

func (s *UnixSambaTestSuite) TestDeleteSambaUser_UserdelFails() {
	username := "sysdeluser"
	userdelActualErr := errors.New("userdel error")
	userdelCmdErr := errors.WithDetails(userdelActualErr, "desc", "command execution failed", "stderr", "userdel critical error")

	mock.When(s.mockCmdExec.RunCommand("smbpasswd", "-x", username)).ThenReturn("", nil).Verify(matchers.Times(1)) // Samba deletion succeeds
	mock.When(s.mockCmdExec.RunCommand("userdel", username)).ThenReturn("", userdelCmdErr).Verify(matchers.Times(1))

	err := unixsamba.DeleteSambaUser(username, true, false)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to delete system user 'sysdeluser'")
	s.True(errors.Is(err, userdelCmdErr))
}

func (s *UnixSambaTestSuite) TestDeleteSambaUser_BothFail_SystemErrorPropagated() {
	username := "bothfailuser"
	smbErrActual := errors.New("smb error")
	smbCmdErr := errors.WithDetails(smbErrActual, "desc", "command execution failed", "stderr", "some other smb error")
	userdelActualErr := errors.New("userdel error")
	userdelCmdErr := errors.WithDetails(userdelActualErr, "desc", "command execution failed", "stderr", "userdel critical error")

	mock.When(s.mockCmdExec.RunCommand("smbpasswd", "-x", username)).ThenReturn("", smbCmdErr).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand("userdel", username)).ThenReturn("", userdelCmdErr).Verify(matchers.Times(1))

	err := unixsamba.DeleteSambaUser(username, true, false)
	s.Require().Error(err)
	// The error message indicates both failures, but the primary returned error is from userdel
	s.Contains(err.Error(), fmt.Sprintf("failed to delete system user '%s' (Samba deletion also failed: %v)", username, smbCmdErr))
	s.True(errors.Is(err, userdelCmdErr)) // The wrapped error is userdelCmdErr
}

// --- ChangePassword Tests ---

func (s *UnixSambaTestSuite) TestChangePassword_SambaOnly_Success() {
	username := "changepwuser"
	newPassword := "newSecurePassword"
	input := newPassword + "\n" + newPassword + "\n"

	mock.When(s.mockCmdExec.RunCommandWithInput(input, "smbpasswd", "-s", username)).ThenReturn("", nil).Verify(matchers.Times(1))

	err := unixsamba.ChangePassword(username, newPassword, true)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestChangePassword_SambaAndSystem_Success() {
	username := "changepwuser"
	newPassword := "newSecurePassword"
	sambaInput := newPassword + "\n" + newPassword + "\n"
	chpasswdInput := username + ":" + newPassword + "\n"

	mock.When(s.mockCmdExec.RunCommandWithInput(sambaInput, "smbpasswd", "-s", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommandWithInput(chpasswdInput, "chpasswd")).ThenReturn("", nil).Verify(matchers.Times(1))

	err := unixsamba.ChangePassword(username, newPassword, false)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestChangePassword_SmbPasswdFails() {
	username := "changepwuser"
	newPassword := "newSecurePassword"
	input := newPassword + "\n" + newPassword + "\n"
	smbErrActual := errors.New("smb error")
	smbCmdErr := errors.WithDetails(smbErrActual, "desc", "command execution with input failed", "stderr", "smb change error")

	mock.When(s.mockCmdExec.RunCommandWithInput(input, "smbpasswd", "-s", username)).ThenReturn("", smbCmdErr).Verify(matchers.Times(1))

	err := unixsamba.ChangePassword(username, newPassword, true)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to change Samba password for user 'changepwuser'")
	s.True(errors.Is(err, smbCmdErr))
}

func (s *UnixSambaTestSuite) TestChangePassword_ChPasswdFails() {
	username := "changepwuser"
	newPassword := "newSecurePassword"
	sambaInput := newPassword + "\n" + newPassword + "\n"
	chpasswdInput := username + ":" + newPassword + "\n"
	sysErrActual := errors.New("sys error")
	sysCmdErr := errors.WithDetails(sysErrActual, "desc", "command execution with input failed", "stderr", "chpasswd error")

	mock.When(s.mockCmdExec.RunCommandWithInput(sambaInput, "smbpasswd", "-s", username)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommandWithInput(chpasswdInput, "chpasswd")).ThenReturn("", sysCmdErr).Verify(matchers.Times(1))

	err := unixsamba.ChangePassword(username, newPassword, false)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to change system password for user 'changepwuser'")
	s.True(errors.Is(err, sysCmdErr))
}

// --- RenameUsername Tests ---

func (s *UnixSambaTestSuite) TestRenameUsername_Success_NoHomeRename() {
	oldUsername := "oldname"
	newUsername := "newname"
	newPassword := "newpass"
	sambaInput := newPassword + "\n" + newPassword + "\n"

	// 1. Check if newUsername exists (system) - should not exist
	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(nil, user.UnknownUserError(newUsername)).Verify(matchers.Times(2))

	// 2. GetByUsername(newUsername) - for Samba check
	//    2a. osUser.Lookup(newUsername) within GetByUsername - should not exist
	//mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(nil, user.UnknownUserError(newUsername)).Verify(matchers.Times(1))
	//    2b. pdbedit for newUsername - should fail with "No such user" or similar
	//pdbeditErr := errors.WithDetails(errors.New("pdbedit failed"), "desc", "command execution failed", "stderr", "No such user newname")
	//mock.When(s.mockCmdExec.RunCommand("pdbedit", "-L", "-v", "-u", newUsername)).ThenReturn("", pdbeditErr).Verify(matchers.Times(1))

	// 3. usermod -l
	mock.When(s.mockCmdExec.RunCommand("usermod", "-l", newUsername, oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1))

	// 4. smbpasswd -x oldname
	mock.When(s.mockCmdExec.RunCommand("smbpasswd", "-x", oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1))

	// 5. smbpasswd -a -s newname
	mock.When(s.mockCmdExec.RunCommandWithInput(sambaInput, "smbpasswd", "-a", "-s", newUsername)).ThenReturn("", nil).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(oldUsername, newUsername, false, newPassword)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestRenameUsername_Success_WithHomeRename() {
	s.T().Skip("Not useful for Rat")
	oldUsername := "oldhome"
	newUsername := "newhome"
	newPassword := "newpass"
	sambaInput := newPassword + "\n" + newPassword + "\n"
	expectedNewHomeDir := "/home/" + newUsername

	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(nil, user.UnknownUserError(newUsername)).Verify(matchers.Times(2)) // Once for initial check, once for GetByUsername
	pdbeditErr := errors.WithDetails(errors.New("pdbedit failed"), "desc", "command execution failed", "stderr", "No such user")
	mock.When(s.mockCmdExec.RunCommand("pdbedit", "-L", "-v", "-u", newUsername)).ThenReturn("", pdbeditErr).Verify(matchers.Times(1))

	mock.When(s.mockCmdExec.RunCommand("usermod", "-l", newUsername, oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1))

	// For home dir rename part
	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(&user.User{Username: newUsername, HomeDir: "/home/" + oldUsername}, nil).Verify(matchers.Times(1)) // After login rename, before home dir rename
	mock.When(s.mockCmdExec.RunCommand("usermod", "-d", expectedNewHomeDir, "-m", newUsername)).ThenReturn("", nil).Verify(matchers.Times(1))

	mock.When(s.mockCmdExec.RunCommand("smbpasswd", "-x", oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommandWithInput(sambaInput, "smbpasswd", "-a", "-s", newUsername)).ThenReturn("", nil).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(oldUsername, newUsername, true, newPassword)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_NewUserSystemExists() {
	oldUsername := "old"
	newUsername := "existing"
	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(&user.User{Username: newUsername}, nil).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(oldUsername, newUsername, false, "pass")
	s.Require().Error(err)
	s.EqualErrorf(err, "new username 'existing' already exists on the system", "")
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_NewUserSambaExists() {
	oldUsername := "old"
	newUsername := "sambanew"

	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(nil, user.UnknownUserError(newUsername)).Verify(matchers.Times(1)) // System check passes

	// GetByUsername for newUsername finds a samba user
	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(&user.User{Username: newUsername}, nil).Verify(matchers.Times(1)) // Inside GetByUsername
	pdbeditOutput := "User SID: S-1-5-blah"
	mock.When(s.mockCmdExec.RunCommand("pdbedit", "-L", "-v", "-u", newUsername)).ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(oldUsername, newUsername, false, "pass")
	s.Require().Error(err)
	s.EqualErrorf(err, "new username 'sambanew' already appears to be a Samba user", "")
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_PdbeditIssueForNewUser() {
	oldUsername := "old"
	newUsername := "pdbissue"

	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(nil, user.UnknownUserError(newUsername)).Verify(matchers.Times(1)) // System check passes

	// GetByUsername for newUsername encounters pdbedit execution error
	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(&user.User{Username: newUsername}, nil).Verify(matchers.Times(1)) // Inside GetByUsername
	pdbeditActualErr := errors.New("pdbedit command failed")
	pdbeditCmdErr := errors.WithDetails(pdbeditActualErr, "desc", "command execution failed", "command", "pdbedit", "stderr", "critical pdbedit error")
	mock.When(s.mockCmdExec.RunCommand("pdbedit", "-L", "-v", "-u", newUsername)).ThenReturn("", pdbeditCmdErr).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(oldUsername, newUsername, false, "pass")
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to verify Samba status for new username 'pdbissue' due to pdbedit execution issue")
	s.True(errors.Is(err, pdbeditCmdErr))
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_UsermodLoginFails() {
	oldUsername := "old"
	newUsername := "new"
	usermodErrActual := errors.New("usermod error")
	usermodCmdErr := errors.WithDetails(usermodErrActual, "desc", "command execution failed", "stderr", "usermod fail")

	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(nil, user.UnknownUserError(newUsername)).Verify(matchers.Times(2))
	pdbeditErr := errors.WithDetails(errors.New("pdbedit failed"), "desc", "command execution failed", "stderr", "No such user")
	mock.When(s.mockCmdExec.RunCommand("pdbedit", "-L", "-v", "-u", newUsername)).ThenReturn("", pdbeditErr).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand("usermod", "-l", newUsername, oldUsername)).ThenReturn("", usermodCmdErr).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(oldUsername, newUsername, false, "pass")
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to rename system user login")
	s.True(errors.Is(err, usermodCmdErr))
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_UsermodHomeFails() {
	s.T().Skip("Not useful for Rat")
	oldUsername := "oldhome"
	newUsername := "newhome"
	newPassword := "newpass"
	expectedNewHomeDir := "/home/" + newUsername
	usermodHomeErrActual := errors.New("usermod home error")
	usermodHomeCmdErr := errors.WithDetails(usermodHomeErrActual, "desc", "command execution failed", "stderr", "usermod home fail")

	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(nil, user.UnknownUserError(newUsername)).Verify(matchers.Times(2))
	pdbeditErr := errors.WithDetails(errors.New("pdbedit failed"), "desc", "command execution failed", "stderr", "No such user")
	mock.When(s.mockCmdExec.RunCommand("pdbedit", "-L", "-v", "-u", newUsername)).ThenReturn("", pdbeditErr).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand("usermod", "-l", newUsername, oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1))

	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(&user.User{Username: newUsername, HomeDir: "/home/" + oldUsername}, nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand("usermod", "-d", expectedNewHomeDir, "-m", newUsername)).ThenReturn("", usermodHomeCmdErr).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(oldUsername, newUsername, true, newPassword)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to move/rename home directory")
	s.True(errors.Is(err, usermodHomeCmdErr))
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_NoPasswordForSamba() {
	err := unixsamba.RenameUsername("old", "new", false, "")
	s.Require().Error(err)
	s.EqualError(err, "a new password must be provided to re-add user to Samba after renaming")
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_SmbPasswdAddFails() {
	oldUsername := "old"
	newUsername := "new"
	newPassword := "newpass"
	sambaInput := newPassword + "\n" + newPassword + "\n"
	smbAddErrActual := errors.New("smb add error")
	smbAddCmdErr := errors.WithDetails(smbAddErrActual, "command execution with input failed", "stderr", "smb add fail")

	mock.When(s.mockOSUser.Lookup(newUsername)).ThenReturn(nil, user.UnknownUserError(newUsername)).Verify(matchers.Times(2))
	pdbeditErr := errors.WithDetails(errors.New("pdbedit failed"), "desc", "command execution failed", "stderr", "No such user")
	mock.When(s.mockCmdExec.RunCommand("pdbedit", "-L", "-v", "-u", newUsername)).ThenReturn("", pdbeditErr).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand("usermod", "-l", newUsername, oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1))
	mock.When(s.mockCmdExec.RunCommand("smbpasswd", "-x", oldUsername)).ThenReturn("", nil).Verify(matchers.Times(1)) // Deletion of old samba user
	mock.When(s.mockCmdExec.RunCommandWithInput(sambaInput, "smbpasswd", "-a", "-s", newUsername)).ThenReturn("", smbAddCmdErr).Verify(matchers.Times(1))

	err := unixsamba.RenameUsername(oldUsername, newUsername, false, newPassword)
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
	mock.When(s.mockCmdExec.RunCommand("pdbedit", "-L")).ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))

	users, err := unixsamba.ListSambaUsers()
	s.Require().NoError(err)
	s.Require().ElementsMatch([]string{"user1", "user2", "adminuser"}, users)
}

func (s *UnixSambaTestSuite) TestListSambaUsers_Success_Empty() {
	pdbeditOutput := ""
	mock.When(s.mockCmdExec.RunCommand("pdbedit", "-L")).ThenReturn(pdbeditOutput, nil).Verify(matchers.Times(1))

	users, err := unixsamba.ListSambaUsers()
	s.Require().NoError(err)
	s.Empty(users)
}

func (s *UnixSambaTestSuite) TestListSambaUsers_PdbeditFails() {
	pdbeditActualErr := errors.New("pdbedit list error")
	pdbeditCmdErr := errors.WithDetails(pdbeditActualErr, "desc", "command execution failed", "stderr", "pdbedit -L failed")

	mock.When(s.mockCmdExec.RunCommand("pdbedit", "-L")).ThenReturn("", pdbeditCmdErr).Verify(matchers.Times(1))

	users, err := unixsamba.ListSambaUsers()
	s.Require().Error(err)
	s.Nil(users)
	s.Contains(err.Error(), "failed to list samba users with pdbedit -L")
	s.True(errors.Is(err, pdbeditCmdErr))
}

/*
// --- Helper for creating errors with details for command execution ---
func newCmdExecError(cmd string, args []string, stderr string, underlyingErr error) error {
	if underlyingErr == nil {
		underlyingErr = errors.New("command execution failed")
	}
	details := []any{"command execution failed", "command", cmd, "args", strings.Join(args, " ")}
	if stderr != "" {
		details = append(details, "stderr", stderr)
	}
	return errors.WithDetails(underlyingErr, details...)
}

func newCmdExecWithInputError(cmd string, args []string, stdin string, stderr string, underlyingErr error) error {
	if underlyingErr == nil {
		underlyingErr = errors.New("command execution with input failed")
	}
	details := []any{"command execution with input failed", "command", cmd, "args", strings.Join(args, " "), "stdin_preview", stdin}
	if stderr != "" {
		details = append(details, "stderr", stderr)
	}
	return errors.WithDetails(underlyingErr, details...)
}
*/
