package unixsamba_test

import (
	"fmt"
	"math/rand/v2"
	"os/user"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dianlight/srat/unixsamba"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
)

type UnixSambaTestSuite struct {
	suite.Suite
	mockSystem *unixsamba.MockSystem
}

func TestUnixSambaTestSuite(t *testing.T) {
	suite.Run(t, new(UnixSambaTestSuite))
}

func (s *UnixSambaTestSuite) SetupTest() {
	s.mockSystem = unixsamba.NewMockSystem()
	unixsamba.SetCommandExecutor(s.mockSystem)
	unixsamba.SetOSUserLookuper(s.mockSystem)
}

func (s *UnixSambaTestSuite) TearDownTest() {
	unixsamba.ResetExecutorsToDefaults()
}

func (s *UnixSambaTestSuite) cmdErr(stderr string) error {
	return errors.WithDetails(
		errors.New("command failed"),
		"desc", "command execution failed",
		"stderr", stderr,
	)
}

func (s *UnixSambaTestSuite) cmdInputErr(stderr string) error {
	return errors.WithDetails(
		errors.New("command failed"),
		"desc", "command execution with input failed",
		"stderr", stderr,
	)
}

func (s *UnixSambaTestSuite) enqueueCmd(command string, args []string, stdout string, err error) {
	s.mockSystem.EnqueueCommandResult(command, args, stdout, err)
}

func (s *UnixSambaTestSuite) enqueueCmdInput(command string, args []string, stdout string, err error) {
	s.mockSystem.EnqueueCommandWithInputResult(command, args, stdout, err)
}

func (s *UnixSambaTestSuite) hasCall(withInput bool, command string, args ...string) bool {
	for _, c := range s.mockSystem.Calls() {
		if c.WithInput != withInput || c.Command != command {
			continue
		}
		if slices.Equal(c.Args, args) {
			return true
		}
	}
	return false
}

// --- GetByUsername Tests ---

func (s *UnixSambaTestSuite) TestGetByUsername_Success_SambaUserExists() {
	username := "testuser"
	s.mockSystem.AddUser(username, "correctpass")
	pdbeditOutput := `
Unix username:        testuser
User SID:             S-1-5-21-xxxx-1001
Primary Group SID:    S-1-5-21-xxxx-513
Password last set:    1679800000
Last logon:           1711447200
	`
	s.enqueueCmd("pdbedit", []string{"-L", "-v", "-u", username}, pdbeditOutput, nil)

	info, err := unixsamba.GetByUsername(s.T().Context(), username)

	s.Require().NoError(err)
	s.Require().NotNil(info)
	s.Equal(username, info.Username)
	s.True(info.IsSambaUser)
	s.Equal("S-1-5-21-xxxx-1001", info.SambaSID)
	s.Equal("S-1-5-21-xxxx-513", info.SambaPrimaryNT)
	s.True(info.SambaPasswordSet)
	s.Equal(time.Unix(1711447200, 0), info.LastLogon)
}

func (s *UnixSambaTestSuite) TestGetByUsername_Success_SambaUserNotExists() {
	username := "testuser"
	s.mockSystem.AddUser(username, "irrelevant")

	pdbeditErr := s.cmdErr("No such user testuser")
	s.enqueueCmd("pdbedit", []string{"-L", "-v", "-u", username}, "", pdbeditErr)

	info, err := unixsamba.GetByUsername(s.T().Context(), username)

	s.Require().NoError(err)
	s.Require().NotNil(info)
	s.False(info.IsSambaUser)
	s.Empty(info.SambaSID)
}

func (s *UnixSambaTestSuite) TestGetByUsername_SystemUserNotFound() {
	username := "nosuchuser"
	lookupErr := user.UnknownUserError(username)
	s.mockSystem.EnqueueLookupResult(username, nil, lookupErr)

	info, err := unixsamba.GetByUsername(s.T().Context(), username)

	s.Require().Error(err)
	s.Nil(info)
	s.True(errors.Is(err, lookupErr))
}

func (s *UnixSambaTestSuite) TestGetByUsername_PdbeditCommandFails() {
	username := "testuser"
	s.mockSystem.AddUser(username, "irrelevant")
	pdbeditCmdErr := s.cmdErr("some other error")
	s.enqueueCmd("pdbedit", []string{"-L", "-v", "-u", username}, "", pdbeditCmdErr)

	info, err := unixsamba.GetByUsername(s.T().Context(), username)

	s.Require().Error(err)
	s.NotNil(info)
	s.False(info.IsSambaUser)
	s.Contains(err.Error(), "pdbedit check for samba user 'testuser' failed")
}

// --- CreateSambaUser Tests ---

func (s *UnixSambaTestSuite) TestCreateSambaUser_Success_NewUser() {
	err := unixsamba.CreateSambaUser(s.T().Context(), "newuser", "password123", unixsamba.UserOptions{CreateHome: true, Shell: "/bin/bash"})
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestCreateSambaUser_Success_SystemUserExists() {
	username := "existinguser"
	s.mockSystem.AddUser(username, "oldpass")
	useraddErr := s.cmdErr("useradd: user 'existinguser' already exists")
	s.enqueueCmd("useradd", []string{"-M", "--badname", username}, "", useraddErr)

	err := unixsamba.CreateSambaUser(s.T().Context(), username, "password123", unixsamba.UserOptions{})
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestCreateSambaUser_UseraddFails_UserNotExists() {
	username := fmt.Sprintf("newuser5%d", rand.IntN(100))
	useraddCmdErr := s.cmdErr("some useradd error")
	s.enqueueCmd("useradd", []string{"-M", "--badname", username}, "", useraddCmdErr)
	s.mockSystem.EnqueueLookupResult(username, nil, errors.New("user not found"))

	err := unixsamba.CreateSambaUser(s.T().Context(), username, "password123", unixsamba.UserOptions{})
	s.Require().Error(err)
	s.Contains(err.Error(), fmt.Sprintf("failed to create system user '%s'", username))
	s.True(errors.Is(err, useraddCmdErr))
}

func (s *UnixSambaTestSuite) TestCreateSambaUser_SmbPasswdFails() {
	username := "newuser"
	smbPasswdCmdErr := s.cmdInputErr("smb error")
	s.enqueueCmdInput("smbpasswd", []string{"-a", "-s", username}, "", smbPasswdCmdErr)

	err := unixsamba.CreateSambaUser(s.T().Context(), username, "password123", unixsamba.UserOptions{})
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to add user 'newuser' to Samba or set password")
	s.True(errors.Is(err, smbPasswdCmdErr))
}

func (s *UnixSambaTestSuite) TestCreateSambaUser_WithOptions() {
	username := "optionsuser"
	password := "securepass"
	options := unixsamba.UserOptions{
		HomeDir:       "/var/customhome",
		Shell:         "/sbin/nologin",
		PrimaryGroup:  "customgroup",
		GECOS:         []string{"group1", "group2"},
		CreateHome:    false,
		SystemAccount: true,
		UID:           "2001",
	}

	err := unixsamba.CreateSambaUser(s.T().Context(), username, password, options)
	s.Require().NoError(err)

	expected := []string{
		"-M", "-r",
		"-d", "/var/customhome",
		"-s", "/sbin/nologin",
		"-G", "customgroup",
		"-g", "group1,group2",
		"-u", "2001",
		"--badname",
		username,
	}
	s.True(s.hasCall(false, "useradd", expected...))
}

// --- DeleteSambaUser Tests ---

func (s *UnixSambaTestSuite) TestDeleteSambaUser_Success() {
	username := "sysdeluser"
	s.mockSystem.AddUser(username, "pass")

	err := unixsamba.DeleteSambaUser(s.T().Context(), username)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestDeleteSambaUser_SmbPasswdFails_NotUserNotFound() {
	username := "smbdeluser"
	s.mockSystem.AddUser(username, "pass")
	smbCmdErr := s.cmdErr("some other smb error")
	s.enqueueCmd("smbpasswd", []string{"-x", username}, "", smbCmdErr)

	err := unixsamba.DeleteSambaUser(s.T().Context(), username)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to delete user 'smbdeluser' from Samba")
	s.True(errors.Is(err, smbCmdErr))
}

func (s *UnixSambaTestSuite) TestDeleteSambaUser_SmbPasswdUserNotFound_SystemDeleteSuccess() {
	username := "smbnotfound"
	s.mockSystem.AddUser(username, "pass")
	smbCmdErr := s.cmdErr("Failed to find entry for user smbnotfound.")
	s.enqueueCmd("smbpasswd", []string{"-x", username}, "", smbCmdErr)

	err := unixsamba.DeleteSambaUser(s.T().Context(), username)
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestDeleteSambaUser_UserdelFails() {
	username := "sysdeluser"
	s.mockSystem.AddUser(username, "pass")
	userdelCmdErr := s.cmdErr("userdel critical error")
	s.enqueueCmd("deluser", []string{"--remove-home", username}, "", userdelCmdErr)

	err := unixsamba.DeleteSambaUser(s.T().Context(), username)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to delete system user 'sysdeluser'")
	s.True(errors.Is(err, userdelCmdErr))
}

// --- ChangePassword Tests ---

func (s *UnixSambaTestSuite) TestChangePassword_Success() {
	username := "changepwuser"
	s.mockSystem.AddUser(username, "oldPassword")

	err := unixsamba.ChangePassword(s.T().Context(), username, "newSecurePassword")
	s.Require().NoError(err)
}

func (s *UnixSambaTestSuite) TestChangePassword_SmbPasswdFails() {
	username := "changepwuser"
	s.mockSystem.AddUser(username, "oldPassword")
	smbCmdErr := s.cmdInputErr("smb change error")
	s.enqueueCmdInput("smbpasswd", []string{"-s", username}, "", smbCmdErr)

	err := unixsamba.ChangePassword(s.T().Context(), username, "newSecurePassword")
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to change Samba password for user 'changepwuser'")
	s.True(errors.Is(err, smbCmdErr))
}

// --- RenameUsername Tests ---

func (s *UnixSambaTestSuite) TestRenameUsername_Success_NoHomeRename() {
	oldUsername := "oldname"
	newUsername := "newname"
	newPassword := "newpass"

	s.mockSystem.AddUser(oldUsername, "oldpass")
	s.mockSystem.EnqueueLookupResult(newUsername, nil, user.UnknownUserError(newUsername))
	s.mockSystem.EnqueueLookupResult(newUsername, &user.User{Username: newUsername, HomeDir: "/home/" + newUsername}, nil)

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, newPassword)
	s.Require().NoError(err)
	s.False(s.hasCall(false, "usermod", "-d", "/home/"+newUsername, "-m", newUsername))
}

func (s *UnixSambaTestSuite) TestRenameUsername_Success_WithHomeRename() {
	oldUsername := "oldhome"
	newUsername := "newhome"
	newPassword := "newpass"

	s.mockSystem.AddUser(oldUsername, "oldpass")

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, newPassword)
	s.Require().NoError(err)
	s.True(s.hasCall(false, "usermod", "-d", "/home/"+newUsername, "-m", newUsername))
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_NewUserSystemExists() {
	s.mockSystem.AddUser("existing", "pass")

	err := unixsamba.RenameUsername(s.T().Context(), "old", "existing", "pass")
	s.Require().Error(err)
	s.EqualError(err, "new username 'existing' already exists on the system")
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_NewUserSambaExists() {
	oldUsername := "old"
	newUsername := "sambanew"

	s.mockSystem.EnqueueLookupResult(newUsername, nil, user.UnknownUserError(newUsername))
	s.enqueueCmd("pdbedit", []string{"-L", "-v", "-u", newUsername}, "User SID: S-1-5-blah", nil)

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, "pass")
	s.Require().Error(err)
	s.EqualError(err, "new username 'sambanew' already appears to be a Samba user")
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_PdbeditIssueForNewUser() {
	oldUsername := "old"
	newUsername := "pdbissue"
	pdbeditCmdErr := s.cmdErr("critical pdbedit error")

	s.mockSystem.EnqueueLookupResult(newUsername, nil, user.UnknownUserError(newUsername))
	s.enqueueCmd("pdbedit", []string{"-L", "-v", "-u", newUsername}, "", pdbeditCmdErr)

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, "pass")
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to verify Samba status for new username 'pdbissue' due to pdbedit execution issue")
	s.True(errors.Is(err, pdbeditCmdErr))
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_UsermodLoginFails() {
	oldUsername := "old"
	newUsername := "new"
	usermodCmdErr := s.cmdErr("usermod fail")

	s.mockSystem.AddUser(oldUsername, "pass")
	s.enqueueCmd("usermod", []string{"-l", newUsername, oldUsername}, "", usermodCmdErr)

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, "pass")
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to rename system user login")
	s.True(errors.Is(err, usermodCmdErr))
}

func (s *UnixSambaTestSuite) TestRenameUsername_Error_UsermodHomeFails() {
	oldUsername := "oldhome"
	newUsername := "newhome"
	usermodHomeCmdErr := s.cmdErr("usermod home fail")

	s.mockSystem.AddUser(oldUsername, "pass")
	s.enqueueCmd("usermod", []string{"-d", "/home/" + newUsername, "-m", newUsername}, "", usermodHomeCmdErr)

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, "newpass")
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
	smbAddCmdErr := s.cmdInputErr("smb add fail")

	s.mockSystem.AddUser(oldUsername, "oldpass")
	s.enqueueCmdInput("smbpasswd", []string{"-a", "-s", newUsername}, "", smbAddCmdErr)

	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, "newpass")
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to add new Samba user 'new' after renaming")
	s.True(errors.Is(err, smbAddCmdErr))
}

// --- ListSambaUsers Tests ---

func (s *UnixSambaTestSuite) TestListSambaUsers_Success() {
	s.mockSystem.AddUser("user1", "pass")
	s.mockSystem.AddUser("user2", "pass")
	s.mockSystem.AddUser("adminuser", "pass")

	users, err := unixsamba.ListSambaUsers(s.T().Context())
	s.Require().NoError(err)
	s.Require().ElementsMatch([]string{"user1", "user2", "adminuser"}, users)
}

func (s *UnixSambaTestSuite) TestListSambaUsers_Success_Empty() {
	users, err := unixsamba.ListSambaUsers(s.T().Context())
	s.Require().NoError(err)
	s.Empty(users)
}

func (s *UnixSambaTestSuite) TestListSambaUsers_PdbeditFails() {
	pdbeditCmdErr := s.cmdErr("pdbedit -L failed")
	s.enqueueCmd("pdbedit", []string{"-L"}, "", pdbeditCmdErr)

	users, err := unixsamba.ListSambaUsers(s.T().Context())
	s.Require().Error(err)
	s.Nil(users)
	s.Contains(err.Error(), "failed to list samba users with pdbedit -L")
	s.True(errors.Is(err, pdbeditCmdErr))
}

// --- CheckSambaUser Tests ---

func (s *UnixSambaTestSuite) TestCheckSambaUser_Success() {
	s.mockSystem.AddUser("activeuser", "correctpass")
	s.Require().NoError(unixsamba.CheckSambaUser(s.T().Context(), "activeuser", "correctpass"))
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_UserNotFound() {
	username := "nosuchsmbuser"
	pdbeditErr := s.cmdErr("Username not found: nosuchsmbuser")
	s.enqueueCmd("pdbedit", []string{"-L", "-v", "-u", username}, "", pdbeditErr)

	err := unixsamba.CheckSambaUser(s.T().Context(), username, "somepass")
	s.Require().Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_PdbeditCommandFails() {
	username := "testuser"
	pdbeditErr := s.cmdErr("unexpected pdbedit error")
	s.enqueueCmd("pdbedit", []string{"-L", "-v", "-u", username}, "", pdbeditErr)

	err := unixsamba.CheckSambaUser(s.T().Context(), username, "somepass")
	s.Require().Error(err)
	s.Contains(err.Error(), "pdbedit check for samba user")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_AccountNotActive() {
	username := "nouflag"
	s.mockSystem.AddUser(username, "somepass")
	s.mockSystem.SetSambaAccountFlags(username, "")

	err := unixsamba.CheckSambaUser(s.T().Context(), username, "somepass")
	s.Require().Error(err)
	s.Contains(err.Error(), "not active")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_AccountDisabled() {
	username := "disableduser"
	s.mockSystem.AddUser(username, "somepass")
	s.mockSystem.SetSambaAccountFlags(username, "DU")

	err := unixsamba.CheckSambaUser(s.T().Context(), username, "somepass")
	s.Require().Error(err)
	s.Contains(err.Error(), "disabled or locked")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_AccountLocked() {
	username := "lockeduser"
	s.mockSystem.AddUser(username, "somepass")
	s.mockSystem.SetSambaAccountFlags(username, "LU")

	err := unixsamba.CheckSambaUser(s.T().Context(), username, "somepass")
	s.Require().Error(err)
	s.Contains(err.Error(), "disabled or locked")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_WrongPassword() {
	s.mockSystem.AddUser("testuser", "correctpass")

	err := unixsamba.CheckSambaUser(s.T().Context(), "testuser", "wrongpass")
	s.Require().Error(err)
	s.Contains(err.Error(), "invalid password")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_PdbeditHashFails() {
	username := "testuser"
	s.mockSystem.AddUser(username, "somepass")
	pdbeditHashErr := s.cmdErr("failed to open database")
	s.enqueueCmd("pdbedit", []string{"-L", "-w", "-u", username}, "", pdbeditHashErr)

	err := unixsamba.CheckSambaUser(s.T().Context(), username, "somepass")
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to read samba password hash")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_NTHashDisabled() {
	username := "testuser"
	s.mockSystem.AddUser(username, "somepass")
	s.mockSystem.SetSambaNTHash(username, strings.Repeat("X", 32))

	err := unixsamba.CheckSambaUser(s.T().Context(), username, "somepass")
	s.Require().Error(err)
	s.Contains(err.Error(), "no password set")
}

func (s *UnixSambaTestSuite) TestCheckSambaUser_NTHashParseError() {
	username := "testuser"
	s.mockSystem.AddUser(username, "somepass")
	s.enqueueCmd("pdbedit", []string{"-L", "-w", "-u", username}, "unrecognised output line", nil)

	err := unixsamba.CheckSambaUser(s.T().Context(), username, "somepass")
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to parse samba password hash")
}
