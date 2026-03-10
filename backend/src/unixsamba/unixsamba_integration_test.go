package unixsamba_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"os/user"
	"testing"

	"github.com/dianlight/srat/unixsamba"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
)

// UnixSambaIntegrationTestSuite tests the unixsamba package using real system commands.
// Each test is automatically skipped when a required command is not found in PATH
// or when the process lacks root privileges for operations that need it.
type UnixSambaIntegrationTestSuite struct {
	suite.Suite
}

func TestUnixSambaIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UnixSambaIntegrationTestSuite))
}

// SetupTest resets to real executors so integration tests are never affected by
// mock state left over from other test suites running in the same binary.
func (s *UnixSambaIntegrationTestSuite) SetupTest() {
	unixsamba.ResetExecutorsToDefaults()
}

// commandAvailable reports whether cmd is found in PATH.
func commandAvailable(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// requireCommands skips the current test if any of the named commands are absent.
func (s *UnixSambaIntegrationTestSuite) requireCommands(cmds ...string) {
	for _, cmd := range cmds {
		if !commandAvailable(cmd) {
			s.T().Skipf("command %q not found in PATH – skipping integration test", cmd)
		}
	}
}

// requireRoot skips the current test unless the process is running as root.
func (s *UnixSambaIntegrationTestSuite) requireRoot() {
	if os.Getuid() != 0 {
		s.T().Skip("test requires root privileges – skipping integration test")
	}
}

// randomTestUsername returns a short, unique username safe for useradd.
func randomTestUsername() string {
	return fmt.Sprintf("srattst%d", rand.IntN(90000)+10000)
}

// cleanupUser is a best-effort helper used in T.Cleanup callbacks.
func cleanupUser(ctx context.Context, username string) {
	err := unixsamba.DeleteSambaUser(ctx, username)
	if err != nil {
		fmt.Printf("cleanupUser: failed to delete user %q: %v\n", username, err)
	}
}

// --- ListSambaUsers ---

func (s *UnixSambaIntegrationTestSuite) TestListSambaUsers_Real() {
	s.requireCommands("pdbedit")
	s.requireRoot()

	users, err := unixsamba.ListSambaUsers(s.T().Context())
	s.Require().NoError(err)
	// Result may be nil (empty database) or a populated slice – both are valid.
	_ = users
}

// --- GetByUsername ---

func (s *UnixSambaIntegrationTestSuite) TestGetByUsername_NonExistentUser_Real() {
	// os/user.Lookup fails before any external command is invoked, so no
	// command availability or root check is needed.
	info, err := unixsamba.GetByUsername(s.T().Context(), "__no_such_srat_user__")
	s.Require().Error(err)
	s.Nil(info)
}

func (s *UnixSambaIntegrationTestSuite) TestGetByUsername_ExistingSystemUser_Real() {
	s.requireCommands("pdbedit")
	s.requireRoot()

	// "root" is guaranteed to exist as a system user in any Linux environment.
	info, err := unixsamba.GetByUsername(s.T().Context(), "root")
	s.Require().NoError(err)
	s.Require().NotNil(info)
	s.Equal("root", info.Username)
	s.Equal("0", info.UID)
}

// --- Full lifecycle: create → inspect → list → delete ---

func (s *UnixSambaIntegrationTestSuite) TestCreateGetDeleteSambaUser_Real() {
	s.requireCommands("useradd", "smbpasswd", "pdbedit", "deluser", "usermod")
	s.requireRoot()

	username := randomTestUsername()
	password := "T3stP@ss!"

	s.T().Cleanup(func() { cleanupUser(s.T().Context(), username) })

	// Create
	errE := unixsamba.CreateSambaUser(s.T().Context(), username, password, unixsamba.UserOptions{
		Shell:      "/sbin/nologin",
		CreateHome: false,
	})
	s.Require().NoError(errE, "CreateSambaUser should succeed for new user",
		"username", username,
		"error", errE,
		"details", errors.AllDetails(errE),
		"wraps", errors.Unwrap(errE),
	)

	// Inspect via GetByUsername
	info, err := unixsamba.GetByUsername(s.T().Context(), username)
	s.Require().NoError(err)
	s.Require().NotNil(info)
	s.Equal(username, info.Username)
	s.True(info.IsSambaUser, "newly created user should be a Samba user")
	s.True(info.SambaPasswordSet, "Samba password should be marked as set")

	// Should appear in ListSambaUsers
	users, err := unixsamba.ListSambaUsers(s.T().Context())
	s.Require().NoError(err)
	s.Contains(users, username)

	// Delete from Samba database only
	err = unixsamba.DeleteSambaUser(s.T().Context(), username)
	s.Require().NoError(err)

	// No longer a Samba user and system user
	info, err = unixsamba.GetByUsername(s.T().Context(), username)
	s.Require().Error(err)

	// System user must be gone
	_, lookupErr := user.Lookup(username)
	s.Require().Error(lookupErr, "system user should have been removed")
}

// --- Create + already exists (idempotent) ---

func (s *UnixSambaIntegrationTestSuite) TestCreateSambaUser_SystemUserAlreadyExists_Real() {
	s.requireCommands("useradd", "smbpasswd", "pdbedit", "deluser", "usermod")
	s.requireRoot()

	username := randomTestUsername()
	password := "T3stP@ss!"

	s.T().Cleanup(func() { cleanupUser(s.T().Context(), username) })

	// First creation
	s.Require().NoError(unixsamba.CreateSambaUser(s.T().Context(), username, password, unixsamba.UserOptions{
		Shell:      "/sbin/nologin",
		CreateHome: false,
	}))

	// Remove only from Samba so the system user still exists
	err := unixsamba.DeleteSambaUser(s.T().Context(), username)
	s.Require().NoError(err)

	// Re-create: system user exists, should be added to Samba without error
	s.Require().NoError(unixsamba.CreateSambaUser(s.T().Context(), username, password, unixsamba.UserOptions{
		Shell:      "/sbin/nologin",
		CreateHome: false,
	}))

	info, err := unixsamba.GetByUsername(s.T().Context(), username)
	s.Require().NoError(err)
	s.True(info.IsSambaUser)
}

func (s *UnixSambaIntegrationTestSuite) TestCreateSambaUser_UsernameWithSpaces_Real() {
	s.requireCommands("useradd", "smbpasswd", "pdbedit", "deluser", "usermod")
	s.requireRoot()

	baseUsername := randomTestUsername()
	usernameWithSpaces := baseUsername[:4] + " " + baseUsername[4:]
	normalizedUsername := unixsamba.NormalizeUsernameForUnixSamba(usernameWithSpaces)
	password := "T3stP@ss!"

	s.T().Cleanup(func() { cleanupUser(s.T().Context(), normalizedUsername) })

	s.Require().NoError(unixsamba.CreateSambaUser(s.T().Context(), usernameWithSpaces, password, unixsamba.UserOptions{
		Shell:      "/sbin/nologin",
		CreateHome: false,
	}))

	info, err := unixsamba.GetByUsername(s.T().Context(), normalizedUsername)
	s.Require().NoError(err)
	s.Require().NotNil(info)
	s.Equal(normalizedUsername, info.Username)
	s.True(info.IsSambaUser)

	users, err := unixsamba.ListSambaUsers(s.T().Context())
	s.Require().NoError(err)
	s.Contains(users, normalizedUsername)
	s.NotContains(users, usernameWithSpaces)
}

func (s *UnixSambaIntegrationTestSuite) TestChangePassword_UsernameWithSpaces_Real() {
	s.requireCommands("useradd", "smbpasswd", "pdbedit", "deluser", "usermod")
	s.requireRoot()

	baseUsername := randomTestUsername()
	usernameWithSpaces := baseUsername[:4] + " " + baseUsername[4:]
	normalizedUsername := unixsamba.NormalizeUsernameForUnixSamba(usernameWithSpaces)
	password := "Init1@lPass!"
	newPassword := "N3wP@ssw0rd!"

	s.T().Cleanup(func() { cleanupUser(s.T().Context(), normalizedUsername) })

	s.Require().NoError(unixsamba.CreateSambaUser(s.T().Context(), usernameWithSpaces, password, unixsamba.UserOptions{
		Shell:      "/sbin/nologin",
		CreateHome: false,
	}))

	err := unixsamba.ChangePassword(s.T().Context(), usernameWithSpaces, newPassword)
	s.Require().NoError(err)

	// Ensure old password no longer works and new password works.
	err = unixsamba.CheckSambaUser(s.T().Context(), normalizedUsername, password)
	s.Require().Error(err)
	err = unixsamba.CheckSambaUser(s.T().Context(), normalizedUsername, newPassword)
	s.Require().NoError(err)
}

func (s *UnixSambaIntegrationTestSuite) TestRenameUsername_WithSpaces_Real() {
	s.requireCommands("useradd", "smbpasswd", "pdbedit", "deluser", "usermod")
	s.requireRoot()

	oldBase := randomTestUsername()
	newBase := randomTestUsername()
	oldUsernameWithSpaces := oldBase[:4] + " " + oldBase[4:]
	newUsernameWithSpaces := newBase[:4] + " " + newBase[4:]
	oldNormalized := unixsamba.NormalizeUsernameForUnixSamba(oldUsernameWithSpaces)
	newNormalized := unixsamba.NormalizeUsernameForUnixSamba(newUsernameWithSpaces)
	password := "Init1@lPass!"
	newPassword := "N3wP@ssw0rd!"

	s.T().Cleanup(func() {
		cleanupUser(s.T().Context(), oldNormalized)
		cleanupUser(s.T().Context(), newNormalized)
	})

	s.Require().NoError(unixsamba.CreateSambaUser(s.T().Context(), oldUsernameWithSpaces, password, unixsamba.UserOptions{
		Shell:      "/sbin/nologin",
		CreateHome: false,
	}))

	err := unixsamba.RenameUsername(s.T().Context(), oldUsernameWithSpaces, newUsernameWithSpaces, newPassword)
	s.Require().NoError(err)

	info, err := unixsamba.GetByUsername(s.T().Context(), newNormalized)
	s.Require().NoError(err)
	s.Require().NotNil(info)
	s.Equal(newNormalized, info.Username)
	s.True(info.IsSambaUser)

	_, lookupErr := user.Lookup(oldNormalized)
	s.Require().Error(lookupErr, "old system user should not exist after rename")

	err = unixsamba.CheckSambaUser(s.T().Context(), newNormalized, newPassword)
	s.Require().NoError(err)
}

func (s *UnixSambaIntegrationTestSuite) TestDeleteSambaUser_UsernameWithSpaces_Real() {
	s.requireCommands("useradd", "smbpasswd", "pdbedit", "deluser", "usermod")
	s.requireRoot()

	baseUsername := randomTestUsername()
	usernameWithSpaces := baseUsername[:4] + " " + baseUsername[4:]
	normalizedUsername := unixsamba.NormalizeUsernameForUnixSamba(usernameWithSpaces)
	password := "T3stP@ss!"

	s.T().Cleanup(func() { cleanupUser(s.T().Context(), normalizedUsername) })

	s.Require().NoError(unixsamba.CreateSambaUser(s.T().Context(), usernameWithSpaces, password, unixsamba.UserOptions{
		Shell:      "/sbin/nologin",
		CreateHome: false,
	}))

	err := unixsamba.DeleteSambaUser(s.T().Context(), usernameWithSpaces)
	s.Require().NoError(err)

	_, getErr := unixsamba.GetByUsername(s.T().Context(), normalizedUsername)
	s.Require().Error(getErr)

	_, lookupErr := user.Lookup(normalizedUsername)
	s.Require().Error(lookupErr, "system user should have been removed")
}

// --- ChangePassword ---

func (s *UnixSambaIntegrationTestSuite) TestChangePassword_SambaOnly_Real() {
	s.requireCommands("useradd", "smbpasswd", "pdbedit", "deluser", "usermod")
	s.requireRoot()

	username := randomTestUsername()
	password := "Init1@lPass!"
	newPassword := "N3wP@ssw0rd!"

	s.T().Cleanup(func() { cleanupUser(s.T().Context(), username) })

	s.Require().NoError(unixsamba.CreateSambaUser(s.T().Context(), username, password, unixsamba.UserOptions{
		Shell:      "/sbin/nologin",
		CreateHome: false,
	}))

	err := unixsamba.ChangePassword(s.T().Context(), username, newPassword)
	s.Require().NoError(err)
}

// --- RenameUsername ---

func (s *UnixSambaIntegrationTestSuite) TestRenameUsername_Real() {
	s.requireCommands("useradd", "smbpasswd", "pdbedit", "deluser", "usermod")
	s.requireRoot()

	oldUsername := randomTestUsername()
	newUsername := randomTestUsername()
	password := "Init1@lPass!"
	newPassword := "N3wP@ssw0rd!"

	s.T().Cleanup(func() {
		cleanupUser(s.T().Context(), oldUsername)
		cleanupUser(s.T().Context(), newUsername)
	})

	// Create the original user
	s.Require().NoError(unixsamba.CreateSambaUser(s.T().Context(), oldUsername, password, unixsamba.UserOptions{
		Shell:      "/sbin/nologin",
		CreateHome: false,
	}))

	// Rename
	err := unixsamba.RenameUsername(s.T().Context(), oldUsername, newUsername, newPassword)
	s.Require().NoError(err)

	// New username must exist as a Samba user
	info, err := unixsamba.GetByUsername(s.T().Context(), newUsername)
	s.Require().NoError(err)
	s.Require().NotNil(info)
	s.Equal(newUsername, info.Username)
	s.True(info.IsSambaUser, "renamed user should be a Samba user")

	// Old system user must no longer exist
	_, lookupErr := user.Lookup(oldUsername)
	s.Require().Error(lookupErr, "old system user should not exist after rename")
}

// --- CheckSambaUser ---

// TestCheckSambaUser_UserNotFound_Real verifies that CheckSambaUser returns an
// error for a username that is not registered in the Samba database.
func (s *UnixSambaIntegrationTestSuite) TestCheckSambaUser_UserNotFound_Real() {
	s.requireCommands("pdbedit")
	s.requireRoot()

	err := unixsamba.CheckSambaUser(s.T().Context(), "__no_such_srat_smb_user__", "irrelevant")
	s.Require().Error(err)
	s.Contains(err.Error(), "not found")
}

// TestCheckSambaUser_FullLifecycle_Real creates a Samba user, verifies the
// correct password is accepted and the wrong password is rejected, then cleans up.
// Verification uses pdbedit NT hash comparison (no smbd required).
func (s *UnixSambaIntegrationTestSuite) TestCheckSambaUser_FullLifecycle_Real() {
	s.requireCommands("useradd", "smbpasswd", "pdbedit", "deluser", "usermod")
	s.requireRoot()

	username := randomTestUsername()
	password := "CheckP@ss1!"
	wrongPassword := "Wr0ngP@ss!"

	s.T().Cleanup(func() { cleanupUser(s.T().Context(), username) })

	// Create the Samba account.
	s.Require().NoError(unixsamba.CreateSambaUser(s.T().Context(), username, password, unixsamba.UserOptions{
		Shell:      "/sbin/nologin",
		CreateHome: false,
	}))

	// Correct password must pass via pdbedit NT hash comparison.
	s.Require().NoError(unixsamba.CheckSambaUser(s.T().Context(), username, password))

	// Wrong password must fail.
	err := unixsamba.CheckSambaUser(s.T().Context(), username, wrongPassword)
	s.Require().Error(err)
}
