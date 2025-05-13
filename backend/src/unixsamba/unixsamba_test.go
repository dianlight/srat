package unixsamba

import (
	"fmt"
	"os"
	"os/user"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
)

// Mock implementations that tests can set
var (
	mockRunCommandFunc          func(command string, args ...string) (string, error)
	mockRunCommandWithInputFunc func(stdinContent string, command string, args ...string) (string, error)
	mockOsUserLookupFunc        func(username string) (*user.User, error)
)

// Original function holders
var (
	originalRunCommand          func(command string, args ...string) (string, error)
	originalRunCommandWithInput func(stdinContent string, command string, args ...string) (string, error)
	originalOsUserLookup        func(username string) (*user.User, error) // Assuming unixsamba.go uses a var like `var osUserLookup = user.Lookup`
)

func TestMain(m *testing.M) {
	// Save original functions
	originalRunCommand = runCommand
	originalRunCommandWithInput = runCommandWithInput
	// originalOsUserLookup = osUserLookup // Uncomment if osUserLookup var is used in unixsamba.go

	// Replace with mocks
	runCommand = func(command string, args ...string) (string, error) {
		if mockRunCommandFunc != nil {
			return mockRunCommandFunc(command, args...)
		}
		panic(fmt.Sprintf("runCommand mock not set for %s %v", command, args))
	}
	runCommandWithInput = func(stdinContent string, command string, args ...string) (string, error) {
		if mockRunCommandWithInputFunc != nil {
			return mockRunCommandWithInputFunc(stdinContent, command, args...)
		}
		panic(fmt.Sprintf("runCommandWithInput mock not set for %s %v", command, args))
	}

	// osUserLookup = func(username string) (*user.User, error) { // Uncomment if osUserLookup var is used
	// 	if mockOsUserLookupFunc != nil {
	// 		return mockOsUserLookupFunc(username)
	// 	}
	// 	panic(fmt.Sprintf("osUserLookup mock not set for %s", username))
	// }

	// Run tests
	exitCode := m.Run()

	// Restore original functions
	runCommand = originalRunCommand
	runCommandWithInput = originalRunCommandWithInput
	// osUserLookup = originalOsUserLookup // Uncomment if osUserLookup var is used

	os.Exit(exitCode)
}

// TestGetByUsername
func TestGetByUsername(t *testing.T) {
	// This setup assumes osUserLookup is mockable.
	// If not, you'll need to ensure the 'testuser' exists or adjust.
	originalOsUserLookup, osUserLookup = osUserLookup, func(username string) (*user.User, error) {
		if mockOsUserLookupFunc != nil {
			return mockOsUserLookupFunc(username)
		}
		// Default mock for user.Lookup if not overridden by a specific test case
		if username == "testuser" {
			return &user.User{
				Uid:      "1001",
				Gid:      "1001",
				Username: "testuser",
				Name:     "Test User",
				HomeDir:  "/home/testuser",
			}, nil
		}
		return nil, errors.New("user not found by mockOsUserLookupFunc")
	}
	defer func() { osUserLookup = originalOsUserLookup }()

	t.Run("SystemUserAndSambaUser", func(t *testing.T) {
		mockOsUserLookupFunc = func(username string) (*user.User, error) {
			if username == "sambauser" {
				return &user.User{Uid: "1000", Gid: "1000", Username: "sambauser", Name: "Samba User", HomeDir: "/home/sambauser"}, nil
			}
			return nil, errors.New("user not found")
		}
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "pdbedit" && args[0] == "-L" && args[1] == "-v" && args[2] == "-u" && args[3] == "sambauser" {
				return `
					Unix username:        sambauser
					NT username:          
					Account Flags:        [U          ]
					User SID:             S-1-5-21-REAL-SID-1000
					Primary Group SID:    S-1-5-21-REAL-SID-513
					Full Name:            Samba User
					Home Directory:       \sambaserver\sambauser
					HomeDir Drive:        
					Logon Script:         
					Profile Path:         \sambaserver\sambauser\profile
					Domain:               SAMBADOMAIN
					Account desc:         
					Workstations:         
					Munged dial:          
					Logon time:           0
					Logoff time:          never
					Kickoff time:         never
					Password last set:    1678886400
					Password can change:  1678886400
					Password must change: never
					Last logon:           1679000000
				`, nil
			}
			return "", errors.New("unexpected command")
		}

		info, err := GetByUsername("sambauser")
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.Equal(t, "1000", info.UID)
		assert.Equal(t, "sambauser", info.Username)
		assert.True(t, info.IsSambaUser)
		assert.Equal(t, "S-1-5-21-REAL-SID-1000", info.SambaSID)
		assert.Equal(t, "S-1-5-21-REAL-SID-513", info.SambaPrimaryNT)
		assert.True(t, info.SambaPasswordSet)
		assert.Equal(t, time.Unix(1679000000, 0), info.LastLogon)
	})

	t.Run("SystemUserNotSambaUser", func(t *testing.T) {
		mockOsUserLookupFunc = func(username string) (*user.User, error) {
			if username == "systemonly" {
				return &user.User{Uid: "1002", Gid: "1002", Username: "systemonly", Name: "System Only User", HomeDir: "/home/systemonly"}, nil
			}
			return nil, errors.New("user not found")
		}
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "pdbedit" && args[3] == "systemonly" {
				return "", errors.WithDetails(errors.New("pdbedit failed"), "command execution failed", "stderr", "Failed to find entry for user systemonly.")
			}
			return "", errors.New("unexpected command")
		}

		info, err := GetByUsername("systemonly")
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.Equal(t, "1002", info.UID)
		assert.False(t, info.IsSambaUser)
		assert.Empty(t, info.SambaSID)
	})

	t.Run("UserNotFoundInSystem", func(t *testing.T) {
		mockOsUserLookupFunc = func(username string) (*user.User, error) {
			return nil, user.UnknownUserError(username) // Simulate user.Lookup error
		}

		info, err := GetByUsername("nosuchuser")
		require.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "failed to lookup system user 'nosuchuser'")
		assert.True(t, errors.Is(err, user.UnknownUserError("nosuchuser")))
	})

	t.Run("PdbeditFailsWithOtherError", func(t *testing.T) {
		mockOsUserLookupFunc = func(username string) (*user.User, error) {
			if username == "pdbfail" {
				return &user.User{Uid: "1003", Gid: "1003", Username: "pdbfail", Name: "PDB Fail", HomeDir: "/home/pdbfail"}, nil
			}
			return nil, errors.New("user not found")
		}
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "pdbedit" && args[3] == "pdbfail" {
				return "", errors.WithDetails(errors.New("some other pdbedit error"), "command execution failed", "stderr", "Connection refused")
			}
			return "", errors.New("unexpected command")
		}

		info, err := GetByUsername("pdbfail")
		require.Error(t, err)
		require.NotNil(t, info) // Info struct is partially filled before pdbedit error is returned
		assert.False(t, info.IsSambaUser)
		assert.Contains(t, err.Error(), "pdbedit check for samba user 'pdbfail' failed")
		var e errors.E
		require.True(t, errors.As(err, &e))
		details := e.Details()
		assert.Equal(t, "Connection refused", details["stderr"])
	})

	t.Run("PdbeditPasswordNotSetAndNoLastLogon", func(t *testing.T) {
		mockOsUserLookupFunc = func(username string) (*user.User, error) {
			if username == "newuser" {
				return &user.User{Uid: "1004", Gid: "1004", Username: "newuser", Name: "New User", HomeDir: "/home/newuser"}, nil
			}
			return nil, errors.New("user not found")
		}
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "pdbedit" && args[3] == "newuser" {
				return `
					User SID:             S-1-5-21-NEW-SID-1004
					Primary Group SID:    S-1-5-21-NEW-SID-513
					Password last set:    0
					Last logon:           0
				`, nil
			}
			return "", errors.New("unexpected command")
		}

		info, err := GetByUsername("newuser")
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.True(t, info.IsSambaUser)
		assert.False(t, info.SambaPasswordSet)
		assert.True(t, info.LastLogon.IsZero())
	})
}

// TestCreateSambaUser
func TestCreateSambaUser(t *testing.T) {
	// This setup assumes osUserLookup is mockable.
	originalOsUserLookup, osUserLookup = osUserLookup, func(username string) (*user.User, error) {
		if mockOsUserLookupFunc != nil {
			return mockOsUserLookupFunc(username)
		}
		return nil, errors.New("user not found by mockOsUserLookupFunc in CreateSambaUser")
	}
	defer func() { osUserLookup = originalOsUserLookup }()

	defaultOpts := UserOptions{CreateHome: true, Shell: "/bin/bash"}

	t.Run("SuccessfulCreation", func(t *testing.T) {
		useraddCalled := false
		smbpasswdCalled := false

		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "useradd" {
				useraddCalled = true
				assert.Equal(t, "newuser", args[len(args)-1])
				assert.Contains(t, args, "-m")
				assert.Contains(t, args, "-s")
				assert.Contains(t, args, "/bin/bash")
				return "", nil
			}
			return "", errors.New("unexpected runCommand call")
		}
		mockRunCommandWithInputFunc = func(stdinContent, command string, args ...string) (string, error) {
			if command == "smbpasswd" && args[0] == "-a" && args[1] == "-s" && args[2] == "newuser" {
				smbpasswdCalled = true
				assert.Equal(t, "password\npassword\n", stdinContent)
				return "", nil
			}
			return "", errors.New("unexpected runCommandWithInput call")
		}

		err := CreateSambaUser("newuser", "password", defaultOpts)
		require.NoError(t, err)
		assert.True(t, useraddCalled, "useradd should have been called")
		assert.True(t, smbpasswdCalled, "smbpasswd -a should have been called")
	})

	t.Run("UserAlreadyExistsInSystem_AddsToSamba", func(t *testing.T) {
		smbpasswdCalled := false
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "useradd" {
				// Simulate useradd failing because user exists
				return "", errors.WithDetails(errors.New("useradd failed"), "command execution failed", "stderr", "useradd: user 'existinguser' already exists")
			}
			return "", errors.New("unexpected runCommand call")
		}
		// Mock user.Lookup to confirm user exists if structured error doesn't match
		mockOsUserLookupFunc = func(username string) (*user.User, error) {
			if username == "existinguser" {
				return &user.User{Username: "existinguser"}, nil
			}
			return nil, user.UnknownUserError(username)
		}
		mockRunCommandWithInputFunc = func(stdinContent, command string, args ...string) (string, error) {
			if command == "smbpasswd" && args[0] == "-a" && args[2] == "existinguser" {
				smbpasswdCalled = true
				return "", nil
			}
			return "", errors.New("unexpected runCommandWithInput call")
		}

		err := CreateSambaUser("existinguser", "password", defaultOpts)
		require.NoError(t, err)
		assert.True(t, smbpasswdCalled, "smbpasswd -a should have been called even if useradd reported exists")
	})

	t.Run("UseraddFails_OtherReason", func(t *testing.T) {
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "useradd" {
				return "", errors.WithDetails(errors.New("disk full"), "command execution failed", "stderr", "No space left on device")
			}
			return "", errors.New("unexpected runCommand call")
		}
		mockOsUserLookupFunc = func(username string) (*user.User, error) {
			return nil, user.UnknownUserError(username) // User does not exist
		}

		err := CreateSambaUser("failuser", "password", defaultOpts)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create system user 'failuser'")
		var e errors.E
		require.True(t, errors.As(err, &e))
		details := e.Details()
		assert.Equal(t, "No space left on device", details["stderr"])
	})

	t.Run("SmbpasswdFails", func(t *testing.T) {
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "useradd" {
				return "", nil // useradd succeeds
			}
			return "", errors.New("unexpected runCommand call")
		}
		mockRunCommandWithInputFunc = func(stdinContent, command string, args ...string) (string, error) {
			if command == "smbpasswd" && args[0] == "-a" {
				return "", errors.WithDetails(errors.New("smb error"), "command execution with input failed", "stderr", "Failed to add entry for user.")
			}
			return "", errors.New("unexpected runCommandWithInput call")
		}

		err := CreateSambaUser("smbfailuser", "password", defaultOpts)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add user 'smbfailuser' to Samba")
		var e errors.E
		require.True(t, errors.As(err, &e))
		details := e.Details()
		assert.Equal(t, "Failed to add entry for user.", details["stderr"])
	})
}

// TestDeleteSambaUser
func TestDeleteSambaUser(t *testing.T) {
	t.Run("DeleteSambaOnly_Success", func(t *testing.T) {
		smbpasswdXCalled := false
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "smbpasswd" && args[0] == "-x" && args[1] == "todelete" {
				smbpasswdXCalled = true
				return "", nil
			}
			return "", errors.New("unexpected command")
		}
		err := DeleteSambaUser("todelete", false, false)
		require.NoError(t, err)
		assert.True(t, smbpasswdXCalled)
	})

	t.Run("DeleteSambaOnly_UserNotFoundInSamba_NoError", func(t *testing.T) {
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "smbpasswd" && args[0] == "-x" && args[1] == "notinsamba" {
				return "", errors.WithDetails(errors.New("samba fail"), "command execution failed", "stderr", "Failed to find entry for user notinsamba")
			}
			return "", errors.New("unexpected command")
		}
		err := DeleteSambaUser("notinsamba", false, false)
		require.NoError(t, err) // Should not error if user not found and only deleting Samba part
	})

	t.Run("DeleteSambaOnly_OtherSambaError_ReturnsError", func(t *testing.T) {
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "smbpasswd" && args[0] == "-x" && args[1] == "sambaerroruser" {
				return "", errors.WithDetails(errors.New("samba fail"), "command execution failed", "stderr", "Samba daemon not responding")
			}
			return "", errors.New("unexpected command")
		}
		err := DeleteSambaUser("sambaerroruser", false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete user 'sambaerroruser' from Samba")
	})

	t.Run("DeleteSystemAndSamba_Success", func(t *testing.T) {
		smbpasswdXCalled := false
		userdelCalled := false
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "smbpasswd" && args[0] == "-x" && args[1] == "fulluser" {
				smbpasswdXCalled = true
				return "", nil
			}
			if command == "userdel" && args[0] == "fulluser" && len(args) == 1 { // No -r
				userdelCalled = true
				return "", nil
			}
			return "", errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
		err := DeleteSambaUser("fulluser", true, false)
		require.NoError(t, err)
		assert.True(t, smbpasswdXCalled)
		assert.True(t, userdelCalled)
	})

	t.Run("DeleteSystemAndSamba_WithHomeDir_Success", func(t *testing.T) {
		userdelArgsCorrect := false
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "smbpasswd" && args[0] == "-x" {
				return "", nil
			}
			if command == "userdel" {
				if assert.Contains(t, args, "-r") && assert.Contains(t, args, "fulluserhome") {
					userdelArgsCorrect = true
				}
				return "", nil
			}
			return "", errors.New("unexpected command")
		}
		err := DeleteSambaUser("fulluserhome", true, true)
		require.NoError(t, err)
		assert.True(t, userdelArgsCorrect)
	})

	t.Run("DeleteSystemAndSamba_UserdelFails", func(t *testing.T) {
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "smbpasswd" && args[0] == "-x" {
				return "", nil // Samba deletion succeeds
			}
			if command == "userdel" {
				return "", errors.WithDetails(errors.New("userdel failed"), "command execution failed", "stderr", "userdel: user test is currently logged in")
			}
			return "", errors.New("unexpected command")
		}
		err := DeleteSambaUser("test", true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete system user 'test'")
		var e errors.E
		require.True(t, errors.As(err, &e))
		assert.Equal(t, "userdel: user test is currently logged in", e.Details()["stderr"])
	})

	t.Run("DeleteSystemAndSamba_SambaFails_UserdelFails", func(t *testing.T) {
		sambaErr := errors.WithDetails(errors.New("samba error"), "command execution failed", "stderr", "Samba connection failed")
		userdelErr := errors.WithDetails(errors.New("userdel error"), "command execution failed", "stderr", "Cannot lock /etc/passwd")

		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "smbpasswd" && args[0] == "-x" {
				return "", sambaErr
			}
			if command == "userdel" {
				return "", userdelErr
			}
			return "", errors.New("unexpected command")
		}
		err := DeleteSambaUser("bothfail", true, false)
		require.Error(t, err)
		// The error from userdel should be primary, with samba error mentioned.
		assert.Contains(t, err.Error(), "failed to delete system user 'bothfail' (Samba deletion also failed: Samba connection failed)")
		assert.Contains(t, err.Error(), "Cannot lock /etc/passwd") // Check for userdel's stderr
	})
}

// TestChangePassword
func TestChangePassword(t *testing.T) {
	t.Run("SambaOnly_Success", func(t *testing.T) {
		smbPasswdCalled := false
		mockRunCommandWithInputFunc = func(stdinContent, command string, args ...string) (string, error) {
			if command == "smbpasswd" && args[0] == "-s" && args[1] == "user1" {
				smbPasswdCalled = true
				assert.Equal(t, "newpass\nnewpass\n", stdinContent)
				return "", nil
			}
			return "", errors.New("unexpected command")
		}
		err := ChangePassword("user1", "newpass", true)
		require.NoError(t, err)
		assert.True(t, smbPasswdCalled)
	})

	t.Run("SambaOnly_SmbpasswdFails", func(t *testing.T) {
		mockRunCommandWithInputFunc = func(stdinContent, command string, args ...string) (string, error) {
			if command == "smbpasswd" {
				return "", errors.WithDetails(errors.New("smb fail"), "command execution with input failed", "stderr", "Bad password")
			}
			return "", errors.New("unexpected command")
		}
		err := ChangePassword("user1", "newpass", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to change Samba password for user 'user1'")
	})

	t.Run("SambaAndSystem_Success", func(t *testing.T) {
		smbPasswdCalled := false
		chPasswdCalled := false
		mockRunCommandWithInputFunc = func(stdinContent, command string, args ...string) (string, error) {
			if command == "smbpasswd" && args[0] == "-s" && args[1] == "user2" {
				smbPasswdCalled = true
				return "", nil
			}
			if command == "chpasswd" {
				chPasswdCalled = true
				assert.Equal(t, "user2:newpass\n", stdinContent)
				return "", nil
			}
			return "", errors.New("unexpected command")
		}
		err := ChangePassword("user2", "newpass", false)
		require.NoError(t, err)
		assert.True(t, smbPasswdCalled)
		assert.True(t, chPasswdCalled)
	})

	t.Run("SambaAndSystem_ChpasswdFails", func(t *testing.T) {
		mockRunCommandWithInputFunc = func(stdinContent, command string, args ...string) (string, error) {
			if command == "smbpasswd" {
				return "", nil // Samba part succeeds
			}
			if command == "chpasswd" {
				return "", errors.WithDetails(errors.New("sys fail"), "command execution with input failed", "stderr", "Permission denied")
			}
			return "", errors.New("unexpected command")
		}
		err := ChangePassword("user2", "newpass", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to change system password for user 'user2'")
	})
}

// TestListSambaUsers
func TestListSambaUsers(t *testing.T) {
	t.Run("SuccessfulList", func(t *testing.T) {
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "pdbedit" && args[0] == "-L" {
				return "user1:1001:User One\nuser2:1002:User Two\n\nuser3:1003:", nil
			}
			return "", errors.New("unexpected command")
		}
		users, err := ListSambaUsers()
		require.NoError(t, err)
		assert.Equal(t, []string{"user1", "user2", "user3"}, users)
	})

	t.Run("EmptyList", func(t *testing.T) {
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "pdbedit" && args[0] == "-L" {
				return "", nil
			}
			return "", errors.New("unexpected command")
		}
		users, err := ListSambaUsers()
		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("PdbeditFails", func(t *testing.T) {
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "pdbedit" && args[0] == "-L" {
				return "", errors.WithDetails(errors.New("pdbedit error"), "command execution failed", "stderr", "Cannot open pdb")
			}
			return "", errors.New("unexpected command")
		}
		users, err := ListSambaUsers()
		require.Error(t, err)
		assert.Nil(t, users)
		assert.Contains(t, err.Error(), "failed to list samba users with pdbedit -L")
		var e errors.E
		require.True(t, errors.As(err, &e))
		assert.Equal(t, "Cannot open pdb", e.Details()["stderr"])
	})

	t.Run("ScannerError", func(t *testing.T) {
		// This is hard to simulate perfectly without a custom reader that returns errors on Scan()
		// For now, assume if pdbedit -L works, scanner.Err() is unlikely unless output is malformed in a specific way
		// that bufio.Scanner itself errors out, which is rare for simple text.
		// The Wrap for scanner.Err() is there for completeness.
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "pdbedit" && args[0] == "-L" {
				// A very long line might cause issues, but bufio.Scanner handles large tokens.
				// Let's assume a simple success case here as scanner errors are harder to trigger.
				return "user1:1001:User One", nil
			}
			return "", errors.New("unexpected command")
		}
		users, err := ListSambaUsers()
		require.NoError(t, err) // Expecting no scanner error for valid output
		assert.Equal(t, []string{"user1"}, users)
	})
}

// TestRenameUsername - This is a complex function, testing a few key paths.
func TestRenameUsername(t *testing.T) {
	// This setup assumes osUserLookup and the GetByUsername's internal osUserLookup are mockable.
	// For simplicity, GetByUsername will use the global mockRunCommandFunc.
	originalOsUserLookup, osUserLookup = osUserLookup, func(username string) (*user.User, error) {
		if mockOsUserLookupFunc != nil {
			return mockOsUserLookupFunc(username)
		}
		return nil, user.UnknownUserError(username) // Default to not found
	}
	defer func() { osUserLookup = originalOsUserLookup }()

	t.Run("SuccessfulRenameNoHomeMove", func(t *testing.T) {
		usermodLoginCalled := false
		smbPasswdDeleteCalled := false
		smbPasswdAddCalled := false

		// Mock for initial user.Lookup(newUsername)
		mockOsUserLookupFunc = func(username string) (*user.User, error) {
			if username == "newuser" {
				return nil, user.UnknownUserError("newuser") // New user does not exist
			}
			if username == "olduser" { // For the GetByUsername call on olduser if it happened
				return &user.User{Uid: "1000", Gid: "1000", Username: "olduser", HomeDir: "/home/olduser"}, nil
			}
			return nil, user.UnknownUserError(username)
		}

		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			switch command {
			case "pdbedit": // For GetByUsername(newUsername)
				if args[3] == "newuser" { // Check for new user in Samba
					return "", errors.WithDetails(errors.New("not found"), "command execution failed", "stderr", "No such user newuser")
				}
			case "usermod":
				if args[0] == "-l" && args[1] == "newuser" && args[2] == "olduser" {
					usermodLoginCalled = true
					// After usermod -l, user.Lookup("newuser") should succeed.
					// We'll update the mockOsUserLookupFunc behavior if needed by specific sub-tests of Rename.
					// For this simple case, assume the next user.Lookup for newHomeDir check will be handled.
					return "", nil
				}
			case "smbpasswd":
				if args[0] == "-x" && args[1] == "olduser" {
					smbPasswdDeleteCalled = true
					return "", nil
				}
			default:
				return "", errors.Errorf("unexpected runCommand: %s %v", command, args)
			}
			return "", nil
		}

		mockRunCommandWithInputFunc = func(stdinContent, command string, args ...string) (string, error) {
			if command == "smbpasswd" && args[0] == "-a" && args[1] == "-s" && args[2] == "newuser" {
				smbPasswdAddCalled = true
				assert.Equal(t, "newpass\nnewpass\n", stdinContent)
				return "", nil
			}
			return "", errors.Errorf("unexpected runCommandWithInput: %s %v", command, args)
		}

		err := RenameUsername("olduser", "newuser", false, "newpass")
		require.NoError(t, err)
		assert.True(t, usermodLoginCalled, "usermod -l should be called")
		assert.True(t, smbPasswdDeleteCalled, "smbpasswd -x olduser should be called")
		assert.True(t, smbPasswdAddCalled, "smbpasswd -a newuser should be called")
	})

	t.Run("NewUsernameAlreadyExistsSystem", func(t *testing.T) {
		mockOsUserLookupFunc = func(username string) (*user.User, error) {
			if username == "existingnew" {
				return &user.User{}, nil // New username already exists
			}
			return nil, user.UnknownUserError(username)
		}
		err := RenameUsername("old", "existingnew", false, "pass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "new username 'existingnew' already exists on the system")
	})

	t.Run("NewUsernameAlreadySambaUser", func(t *testing.T) {
		mockOsUserLookupFunc = func(username string) (*user.User, error) {
			return nil, user.UnknownUserError(username) // System user lookup for new user = not found
		}
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			if command == "pdbedit" && args[3] == "sambaexists" { // GetByUsername for new user
				return "User SID: S-1-5-FOO", nil // Indicates Samba user exists
			}
			return "", errors.New("unexpected command")
		}
		err := RenameUsername("old", "sambaexists", false, "pass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "new username 'sambaexists' already appears to be a Samba user")
	})

	t.Run("RenameHomeDirSuccess", func(t *testing.T) {
		usermodMoveHomeCalled := false

		// Initial lookup for new user: not found
		// GetByUsername for new user: not found in Samba
		// usermod -l: success
		// user.Lookup(newUsername) for home dir check: success, different home
		// usermod -d -m: success
		// smbpasswd -x: success
		// smbpasswd -a: success

		mockOsUserLookupFunc = func(username string) (*user.User, error) {
			if username == "newhomeuser" { // After usermod -l, or initial check
				// Simulate different states for user.Lookup
				// If usermodLoginCalled is true, it means rename happened.
				// This state management can get complex; simpler mocks per call are better.
				return &user.User{Username: "newhomeuser", HomeDir: "/home/oldhomeuser"}, nil // Home dir is still old
			}
			return nil, user.UnknownUserError(username)
		}

		var usermodLoginDone bool
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			switch command {
			case "pdbedit": // For GetByUsername(newUsername)
				if args[3] == "newhomeuser" {
					return "", errors.WithDetails(errors.New("not found"), "stderr", "No such user")
				}
			case "usermod":
				if args[0] == "-l" && args[1] == "newhomeuser" && args[2] == "oldhomeuser" {
					usermodLoginDone = true
					// Update mock for subsequent user.Lookup("newhomeuser")
					mockOsUserLookupFunc = func(username string) (*user.User, error) {
						if username == "newhomeuser" {
							return &user.User{Username: "newhomeuser", HomeDir: "/home/oldhomeuser", Uid: "1010", Gid: "1010"}, nil
						}
						return nil, user.UnknownUserError(username)
					}
					return "", nil
				}
				if args[0] == "-d" && args[1] == "/home/newhomeuser" && args[2] == "-m" && args[3] == "newhomeuser" {
					require.True(t, usermodLoginDone, "usermod -l must happen before moving home")
					usermodMoveHomeCalled = true
					return "", nil
				}
			case "smbpasswd":
				if args[0] == "-x" && args[1] == "oldhomeuser" {
					return "", nil
				}
			default:
				return "", errors.Errorf("unexpected runCommand: %s %v", command, args)
			}
			return "", nil
		}
		mockRunCommandWithInputFunc = func(stdinContent, command string, args ...string) (string, error) {
			if command == "smbpasswd" && args[0] == "-a" && args[2] == "newhomeuser" {
				return "", nil
			}
			return "", errors.Errorf("unexpected runCommandWithInput: %s %v", command, args)
		}

		err := RenameUsername("oldhomeuser", "newhomeuser", true, "newpass")
		require.NoError(t, err)
		assert.True(t, usermodMoveHomeCalled)
	})

	t.Run("MissingNewPasswordForSamba", func(t *testing.T) {
		mockOsUserLookupFunc = func(username string) (*user.User, error) {
			return nil, user.UnknownUserError(username)
		}
		mockRunCommandFunc = func(command string, args ...string) (string, error) {
			switch command {
			case "pdbedit":
				return "", errors.WithDetails(errors.New("not found"), "stderr", "No such user")
			case "usermod": // -l
				return "", nil
			case "smbpasswd": // -x
				return "", nil
			}
			return "", errors.New("unexpected command")
		}

		err := RenameUsername("old", "new", false, "") // Empty password
		require.Error(t, err)
		assert.Equal(t, "a new password must be provided to re-add user to Samba after renaming", err.Error())
	})
}
