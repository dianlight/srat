package unixsamba

import (
	"bufio"
	"bytes"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"time"

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
		return stdout, errors.WithDetails(err, "command execution with input failed",
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
			if stderr, ok := details["stderr"].(string); ok && (strings.Contains(stderr, "No such user") || strings.Contains(stderr, "does not exist")) {
				info.IsSambaUser = false // Explicitly set, though it's the default
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
func CreateSambaUser(username string, password string, options UserOptions) error {
	useraddArgs := []string{}
	if !options.CreateHome {
		useraddArgs = append(useraddArgs, "-H")
	}
	if options.SystemAccount {
		useraddArgs = append(useraddArgs, "-S")
	}
	if options.HomeDir != "" {
		useraddArgs = append(useraddArgs, "-h", options.HomeDir)
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
	useraddArgs = append(useraddArgs, "-D") //  Don't assign a password
	useraddArgs = append(useraddArgs, username)

	_, err := cmdExec.RunCommand("adduser", useraddArgs...)
	if err != nil {
		// Check if the error is because the user already exists
		var e errors.E
		userExists := false
		if errors.As(err, &e) {
			details := e.Details()
			if stderr, ok := details["stderr"].(string); ok && strings.Contains(strings.ToLower(stderr), "useradd: user '"+strings.ToLower(username)+"' already exists") {
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
			return errors.Wrapf(err, "failed to create system user '%s'", username)
		}
		// If user exists, we can proceed to add to Samba (log this if logger available)
	}

	smbPasswdInput := password + "\n" + password + "\n"
	_, err = cmdExec.RunCommandWithInput(smbPasswdInput, "smbpasswd", "-a", "-s", username)
	if err != nil {
		return errors.Wrapf(err, "failed to add user '%s' to Samba or set password", username)
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
			userdelArgs = append(userdelArgs, "-r")
		}
		userdelArgs = append(userdelArgs, username)
		_, sysErr := cmdExec.RunCommand("userdel", userdelArgs...)
		if sysErr != nil {
			// If Samba deletion also failed (and it wasn't "user not found")
			if !sambaUserDeleted {
				return errors.Wrapf(sysErr, "failed to delete system user '%s' (Samba deletion also failed: %v)", username, err)
			}
			return errors.Wrapf(sysErr, "failed to delete system user '%s'", username)
		}
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
		// Log or handle, but proceed to add new user
		// errors.Wrapf(delErr, "failed to delete old Samba user '%s'", oldUsername)
	}

	if newPasswordForSamba == "" {
		return errors.New("a new password must be provided to re-add user to Samba after renaming")
	}
	smbPasswdInput := newPasswordForSamba + "\n" + newPasswordForSamba + "\n"
	_, addErr := cmdExec.RunCommandWithInput(smbPasswdInput, "smbpasswd", "-a", "-s", newUsername)
	if addErr != nil {
		return errors.Wrapf(addErr, "failed to add new Samba user '%s' after renaming", newUsername)
	}
	return nil
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
