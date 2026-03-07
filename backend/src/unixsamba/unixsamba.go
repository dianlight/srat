package unixsamba

import (
	"bufio"
	"bytes"
	"log/slog"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/dianlight/srat/internal/osutil"
	"gitlab.com/tozd/go/errors" // Import the new errors package
)

// UserInfo holds information about a Unix and potentially Samba user.
type UserInfo struct {
	UID              string
	GID              string
	Username         string
	Name             string // GECOS field
	HomeDir          string
	Shell            string
	IsSambaUser      bool
	SambaSID         string
	SambaPrimaryNT   string // Primary Group SID or RID for Samba
	SambaPasswordSet bool   // Indicates if the Samba password is set
	LastLogon        time.Time
}

// CommandExecutor defines an interface for running external commands.
type CommandExecutor interface {
	RunCommand(command string, args ...string) (string, error)
	RunCommandWithInput(stdinContent string, command string, args ...string) (string, error)
}

// OSUserLookuper defines an interface for looking up OS users.
type OSUserLookuper interface {
	Lookup(username string) (*user.User, error)
}

// defaultCommandExecutor implements CommandExecutor using os/exec.
type defaultCommandExecutor struct{}

// defaultOSUserLookuper implements OSUserLookuper using os/user.
type defaultOSUserLookuper struct{}

// Package-level variables for holding the implementations.
var cmdExec CommandExecutor = &defaultCommandExecutor{}
var osUser OSUserLookuper = &defaultOSUserLookuper{}

// SetCommandExecutor allows overriding the default command executor for testing.
func SetCommandExecutor(executor CommandExecutor) {
	cmdExec = executor
}

// SetOSUserLookuper allows overriding the default OS user lookuper for testing.
func SetOSUserLookuper(lookuper OSUserLookuper) {
	osUser = lookuper
}

// ResetExecutorsToDefaults restores the default command executor and OS user lookuper.
// This is primarily intended for use in test cleanup.
func ResetExecutorsToDefaults() {
	cmdExec = &defaultCommandExecutor{}
	osUser = &defaultOSUserLookuper{}
}

// UserOptions specifies parameters for creating a new system user.
type UserOptions struct {
	HomeDir       string
	Shell         string
	PrimaryGroup  string
	GECOS         []string
	CreateHome    bool
	SystemAccount bool
	UID           string
	GID           string
}

// RunCommand is the actual implementation for running commands.
func (d *defaultCommandExecutor) RunCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	var outData bytes.Buffer
	var stderrData bytes.Buffer
	cmd.Stdout = &outData
	cmd.Stderr = &stderrData

	err := cmd.Run()
	stdout := outData.String()
	stderr := stderrData.String()

	if err != nil {
		// Use errors.Errorf for structured error information
		return stdout, errors.WithDetails(err, "desc", "command execution failed",
			"command", command,
			"args", args,
			"stderr", stderr,
			"stdout", stdout,
		)
	}
	return stdout, nil
}

// RunCommandWithInput is the actual implementation for running commands with stdin.
func (d *defaultCommandExecutor) RunCommandWithInput(stdinContent string, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Stdin = strings.NewReader(stdinContent)
	var outData bytes.Buffer
	var stderrData bytes.Buffer
	cmd.Stdout = &outData
	cmd.Stderr = &stderrData

	err := cmd.Run()
	stdout := outData.String()
	stderr := stderrData.String()

	if err != nil {
		return stdout, errors.WithDetails(err, "desc", "command execution with input failed",
			"command", command,
			"args", args,
			"stdin_preview", func() string {
				if len(stdinContent) > 50 {
					return stdinContent[:50] + "..."
				}
				return stdinContent
			}(),
			"stderr", stderr,
			"stdout", stdout,
		)
	}
	return stdout, nil
}

// Lookup is the actual implementation for user lookup.
func (d *defaultOSUserLookuper) Lookup(username string) (*user.User, error) {
	return user.Lookup(username)
}

// GetByUsername retrieves information about a Unix user and checks their Samba status.
func GetByUsername(username string) (*UserInfo, error) {
	sysUser, err := osUser.Lookup(username)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to lookup system user '%s'", username)
	}

	info := &UserInfo{
		UID:         sysUser.Uid,
		GID:         sysUser.Gid,
		Username:    sysUser.Username,
		Name:        sysUser.Name,
		HomeDir:     sysUser.HomeDir,
		IsSambaUser: false,
	}

	pdbeditOutput, err := cmdExec.RunCommand("pdbedit", "-L", "-v", "-u", username)
	if err == nil {
		info.IsSambaUser = true
		scanner := bufio.NewScanner(strings.NewReader(pdbeditOutput))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				switch key {
				case "User SID":
					info.SambaSID = value
				case "Primary Group SID":
					info.SambaPrimaryNT = value
				case "Password last set":
					if value != "0" && !strings.Contains(strings.ToLower(value), "never") && value != "" {
						info.SambaPasswordSet = true
					}
				case "Last logon":
					if valInt, convErr := strconv.ParseInt(value, 10, 64); convErr == nil && valInt > 0 {
						info.LastLogon = time.Unix(valInt, 0)
					}
				}
			}
		}
	} else {
		var e errors.E
		if errors.As(err, &e) {
			details := e.Details()
			if stderr, ok := details["stderr"].(string); ok {
				lwrStderr := strings.ToLower(stderr)
				if strings.Contains(lwrStderr, "no such user") ||
					strings.Contains(lwrStderr, "does not exist") ||
					strings.Contains(lwrStderr, "username not found") ||
					strings.Contains(lwrStderr, "user not found") {
					info.IsSambaUser = false // Explicitly set, though it's the default
				} else {
					return info, errors.Wrapf(err, "pdbedit check for samba user '%s' failed", username)
				}
			} else {
				return info, errors.Wrapf(err, "pdbedit check for samba user '%s' failed", username)
			}
		} else {
			return info, errors.Wrapf(err, "pdbedit check for samba user '%s' failed (unknown error type)", username)
		}
	}
	return info, nil
}

// CreateSambaUser creates a system Unix user and then adds them to the Samba database.
func CreateSambaUser(username string, password string, options UserOptions) errors.E {
	useraddArgs := []string{}
	if !options.CreateHome {
		useraddArgs = append(useraddArgs, "-M")
	}
	if options.SystemAccount {
		useraddArgs = append(useraddArgs, "-r")
	}
	if options.HomeDir != "" {
		useraddArgs = append(useraddArgs, "-d", options.HomeDir)
	}
	if options.Shell != "" {
		useraddArgs = append(useraddArgs, "-s", options.Shell)
	}
	if options.PrimaryGroup != "" {
		useraddArgs = append(useraddArgs, "-G", options.PrimaryGroup)
	}
	if len(options.GECOS) > 0 {
		useraddArgs = append(useraddArgs, "-g", strings.Join(options.GECOS, ","))
	}
	if options.UID != "" {
		useraddArgs = append(useraddArgs, "-u", options.UID)
	}
	useraddArgs = append(useraddArgs, "--badname") //  do not check for bad names
	useraddArgs = append(useraddArgs, username)

	_, err := cmdExec.RunCommand("useradd", useraddArgs...)
	if err != nil {
		// Check if the error is because the user already exists
		var e errors.E
		userExists := false
		if errors.As(err, &e) {
			details := e.Details()
			if stderr, ok := details["stderr"].(string); ok && strings.Contains(strings.ToLower(stderr), "useradd: user "+strings.ToLower(username)+" already exists") {
				userExists = true
			}
		}
		// Fallback check using os/user if structured error didn't confirm
		if !userExists {
			if _, lookupErr := osUser.Lookup(username); lookupErr == nil {
				userExists = true
			}
		}

		if !userExists {
			return errors.WithMessagef(err, "failed to create system user '%s'", username)
		}
		// If user exists, we can proceed to add to Samba (log this if logger available)
	}

	smbPasswdInput := password + "\n" + password + "\n"
	_, err = cmdExec.RunCommandWithInput(smbPasswdInput, "smbpasswd", "-a", "-s", username)
	if err != nil {
		return errors.Errorf("failed to add user '%s' to Samba or set password %w", username, err)
	}

	// Use CheckSambaUser to verify the new Samba user is functional with the new password before confirming success. This also serves as a sanity check that the user can authenticate with Samba after the rename.
	if err := CheckSambaUser(username, password); err != nil {
		return errors.Wrapf(err, "verification of Samba user '%s' failed after creation", username)
	}
	return nil
}

// DeleteSambaUser deletes a user from Samba and optionally from the system.
func DeleteSambaUser(username string, deleteSystemUser bool, deleteHomeDir bool) error {
	_, err := cmdExec.RunCommand("smbpasswd", "-x", username)
	sambaUserDeleted := err == nil

	if err != nil {
		// Check if the error is "user not found" which is not fatal if we also want to delete system user
		isUserNotFoundErr := false
		var e errors.E
		if errors.As(err, &e) {
			details := e.Details()
			if stderr, ok := details["stderr"].(string); ok {
				lwrStderr := strings.ToLower(stderr)
				if strings.Contains(lwrStderr, "failed to find entry for user") || strings.Contains(lwrStderr, "no such user") {
					isUserNotFoundErr = true
				}
			}
		}

		if !isUserNotFoundErr && !deleteSystemUser { // If it's another error and we are ONLY deleting samba user
			return errors.Wrapf(err, "failed to delete user '%s' from Samba", username)
		}
		// If isUserNotFoundErr, we can proceed to system user deletion without erroring here.
	}

	if deleteSystemUser {
		userdelArgs := []string{}
		if deleteHomeDir {
			userdelArgs = append(userdelArgs, "--remove-home")
		}
		userdelArgs = append(userdelArgs, username)
		_, sysErr := cmdExec.RunCommand("deluser", userdelArgs...)
		if sysErr != nil {
			// If Samba deletion also failed (and it wasn't "user not found")
			if !sambaUserDeleted {
				return errors.Wrapf(sysErr, "failed to delete system user '%s' (Samba deletion also failed: %v)", username, err)
			}
			return errors.Wrapf(sysErr, "failed to delete system user '%s'", username)
		}
	}

	// Use CheckSambaUser to verify the new Samba user is deleted. If the user still exists in Samba, this will return an error which we can log but not fail on since the main deletion logic has already been attempted.
	if err := CheckSambaUser(username, "invalidpasswordforsure"); err == nil {
		slog.Warn("User still appears to exist in Samba after deletion attempt", "username", username)
	}

	return nil
}

// ChangePassword changes a user's Samba password and optionally their system password.
func ChangePassword(username string, newPassword string, sambaOnly bool) error {
	smbPasswdInput := newPassword + "\n" + newPassword + "\n"
	_, err := cmdExec.RunCommandWithInput(smbPasswdInput, "smbpasswd", "-s", username)
	if err != nil {
		return errors.Wrapf(err, "failed to change Samba password for user '%s'", username)
	}

	if !sambaOnly {
		chpasswdInput := username + ":" + newPassword + "\n"
		_, sysErr := cmdExec.RunCommandWithInput(chpasswdInput, "chpasswd")
		if sysErr != nil {
			return errors.Wrapf(sysErr, "failed to change system password for user '%s'", username)
		}
	}

	// Use CheckSambaUser to verify the new Samba user is functional with the new password before confirming success. This also serves as a sanity check that the user can authenticate with Samba after the rename.
	if err := CheckSambaUser(username, newPassword); err != nil {
		return errors.Wrapf(err, "verification of Samba user '%s' failed after password change", username)
	}
	return nil
}

// RenameUsername renames a Unix system user and attempts to reflect this in Samba.
// WARNING: This will likely change the user's Samba SID.
func RenameUsername(oldUsername string, newUsername string, renameHomeDir bool, newPasswordForSamba string) error {
	if _, err := osUser.Lookup(newUsername); err == nil {
		return errors.Errorf("new username '%s' already exists on the system", newUsername)
	}

	sambaInfo, sambaErr := GetByUsername(newUsername)
	if sambaErr == nil && sambaInfo.IsSambaUser { // User found and is a samba user
		return errors.Errorf("new username '%s' already appears to be a Samba user", newUsername)
	} else if sambaErr != nil { // GetByUsername returned an error
		// Check if it's a pdbedit execution error (not just "user not found" for samba part)
		var e errors.E
		isPdbeditIssue := false
		if errors.As(sambaErr, &e) {
			details := e.Details()
			// Check if the *original* error for pdbedit (before GetByUsername wrapped it) was more than just user not found
			// This is tricky as GetByUsername already filters "No such user" for the Samba part.
			// Any error from GetByUsername at this stage implies a more fundamental issue if it's not a system user lookup error
			// (which should have been caught by the first user.Lookup(newUsername)).
			// So, if sambaErr exists, it means either the system lookup within GetByUsername failed (unlikely here)
			// or the pdbedit command itself had an execution issue beyond just user not found.
			if cmd, ok := details["command"].(string); ok && cmd == "pdbedit" {
				isPdbeditIssue = true // Indicates a failure in running pdbedit itself.
			}
		}
		if isPdbeditIssue {
			return errors.Wrapf(sambaErr, "failed to verify Samba status for new username '%s' due to pdbedit execution issue", newUsername)
		}
		// If not a pdbedit issue (e.g. system user not found in GetByUsername, which is good here), we can proceed.
	}

	usermodArgs := []string{"-l", newUsername, oldUsername}
	_, err := cmdExec.RunCommand("usermod", usermodArgs...)
	if err != nil {
		return errors.Wrapf(err, "failed to rename system user login from '%s' to '%s'", oldUsername, newUsername)
	}

	if renameHomeDir {
		currentSysUser, lookupErr := osUser.Lookup(newUsername)
		if lookupErr != nil {
			return errors.Wrapf(lookupErr, "failed to lookup new system user '%s' after rename", newUsername)
		}
		newHomeDir := "/home/" + newUsername
		if currentSysUser.HomeDir != newHomeDir {
			_, err = cmdExec.RunCommand("usermod", "-d", newHomeDir, "-m", newUsername)
			if err != nil {
				return errors.Wrapf(err, "failed to move/rename home directory to '%s' for user '%s'", newHomeDir, newUsername)
			}
		}
	}

	_, delErr := cmdExec.RunCommand("smbpasswd", "-x", oldUsername)
	if delErr != nil {
		slog.Error("Unable to delete old Samba user", "error", delErr, "username", oldUsername)
	}

	if newPasswordForSamba == "" {
		return errors.New("a new password must be provided to re-add user to Samba after renaming")
	}
	smbPasswdInput := newPasswordForSamba + "\n" + newPasswordForSamba + "\n"
	_, addErr := cmdExec.RunCommandWithInput(smbPasswdInput, "smbpasswd", "-a", "-s", newUsername)
	if addErr != nil {
		return errors.Wrapf(addErr, "failed to add new Samba user '%s' after renaming", newUsername)
	}

	// Use CheckSambaUser to verify the new Samba user is functional with the new password before confirming success. This also serves as a sanity check that the user can authenticate with Samba after the rename.
	if err := CheckSambaUser(newUsername, newPasswordForSamba); err != nil {
		return errors.Wrapf(err, "verification of new Samba user '%s' failed after renaming", newUsername)
	}

	return nil
}

// CheckSambaUser verifies that username exists as an active Samba user and that
// password is correct.
//
// Steps:
//  1. pdbedit -L -v confirms the account exists and the Account Flags contain
//     [U] (active). Accounts flagged [D] (disabled) or [L] (locked) are rejected.
//  2. pdbedit -L -w extracts the stored NT password hash (smbpasswd format).
//     The NT hash of the supplied password is computed in pure Go (MD4 of the
//     UTF-16LE-encoded password) and compared with the stored value.
//     This approach works regardless of whether smbd is running.
func CheckSambaUser(username, password string) error {
	// Step 1: Confirm the user exists in the Samba database and is active.
	pdbeditOutput, err := cmdExec.RunCommand("pdbedit", "-L", "-v", "-u", username)
	if err != nil {
		var e errors.E
		if errors.As(err, &e) {
			if stderr, ok := e.Details()["stderr"].(string); ok {
				lwr := strings.ToLower(stderr)
				if strings.Contains(lwr, "no such user") ||
					strings.Contains(lwr, "username not found") ||
					strings.Contains(lwr, "user not found") ||
					strings.Contains(lwr, "does not exist") {
					return errors.Errorf("samba user '%s' not found", username)
				}
			}
		}
		return errors.Wrapf(err, "pdbedit check for samba user '%s' failed", username)
	}

	// Parse Account Flags from pdbedit output.
	accountFlags := ""
	scanner := bufio.NewScanner(strings.NewReader(pdbeditOutput))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == "Account Flags" {
			accountFlags = strings.TrimSpace(parts[1])
			break
		}
	}

	if !strings.Contains(accountFlags, "U") {
		return errors.Errorf("samba user '%s' account is not active (flags: %s)", username, accountFlags)
	}
	if strings.Contains(accountFlags, "D") || strings.Contains(accountFlags, "L") {
		return errors.Errorf("samba user '%s' account is disabled or locked (flags: %s)", username, accountFlags)
	}

	// Step 2: Extract the stored NT hash via pdbedit smbpasswd format and
	// compare it with the NT hash of the supplied password.
	smbPasswdOut, err := cmdExec.RunCommand("pdbedit", "-L", "-w", "-u", username)
	if err != nil {
		return errors.Wrapf(err, "failed to read samba password hash for user '%s'", username)
	}

	storedNTHash, err := parseSmbPasswdNTHash(username, smbPasswdOut)
	if err != nil {
		return errors.Wrapf(err, "failed to parse samba password hash for user '%s'", username)
	}

	// A hash of 32 'X' characters means no password is set.
	if isSmbDisabledHash(storedNTHash) {
		return errors.Errorf("samba user '%s' has no password set", username)
	}

	if !strings.EqualFold(osutil.NTHash(password), storedNTHash) {
		return errors.Errorf("invalid password for samba user '%s'", username)
	}
	return nil
}

// parseSmbPasswdNTHash extracts the NT hash field from a pdbedit -L -w output
// line.  The smbpasswd format is: username:uid:LMHASH:NTHASH:flags:::
func parseSmbPasswdNTHash(username, output string) (string, error) {
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, username+":") {
			continue
		}
		parts := strings.SplitN(line, ":", 5)
		if len(parts) < 4 {
			continue
		}
		nt := strings.ToUpper(parts[3])
		if len(nt) != 32 {
			return "", errors.Errorf("unexpected NT hash length %d in pdbedit smbpasswd output", len(nt))
		}
		return nt, nil
	}
	return "", errors.New("NT hash line not found in pdbedit smbpasswd output")
}

// isSmbDisabledHash reports whether the hash string represents a disabled or
// unset Samba password (all 32 characters are 'X').
func isSmbDisabledHash(hash string) bool {
	if len(hash) != 32 {
		return false
	}
	for _, c := range hash {
		if c != 'X' {
			return false
		}
	}
	return true
}

// ListSambaUsers retrieves a list of all usernames known to Samba.
// This function requires privileges to run `pdbedit -L`.
func ListSambaUsers() ([]string, error) {
	output, err := cmdExec.RunCommand("pdbedit", "-L")
	if err != nil {
		return nil, errors.Wrap(err, "failed to list samba users with pdbedit -L")
	}

	var users []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Default `pdbedit -L` output is often `username:UID:FullName` or similar.
		// We only need the username part.
		parts := strings.SplitN(line, ":", 2)
		if len(parts) > 0 && parts[0] != "" {
			users = append(users, parts[0])
		}
	}

	if err := scanner.Err(); err != nil {
		return users, errors.Wrap(err, "failed to scan pdbedit output")
	}

	return users, nil
}
