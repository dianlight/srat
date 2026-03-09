package unixsamba

import (
	"context"
	"fmt"
	"os/user"
	"slices"
	"strings"
	"sync"

	"github.com/dianlight/srat/internal/osutil"
	"gitlab.com/tozd/go/errors"
)

type mockUserState struct {
	osUser       *user.User
	accountFlags string
	ntHash       string
	hasSamba     bool
}

type mockCommandResponse struct {
	stdout string
	err    error
}

// MockCommandCall captures a command invocation made through MockSystem.
type MockCommandCall struct {
	WithInput bool
	Command   string
	Args      []string
	Stdin     string
}

// MockSystem is a generic in-memory mock implementation for both
// CommandExecutor and OSUserLookuper.
//
// It is safe for concurrent use and intended for tests that need predictable,
// stateful unix/samba behavior without invoking external commands.
type MockSystem struct {
	mu sync.RWMutex

	users   map[string]*mockUserState
	nextUID int

	commandQueue      map[string][]mockCommandResponse
	commandInputQueue map[string][]mockCommandResponse
	lookupQueue       map[string][]*queuedLookup
	calls             []MockCommandCall
}

type queuedLookup struct {
	osUser *user.User
	err    error
}

// NewMockSystem builds a new empty mock unix/samba system.
func NewMockSystem() *MockSystem {
	return &MockSystem{
		users:             make(map[string]*mockUserState),
		nextUID:           1000,
		commandQueue:      make(map[string][]mockCommandResponse),
		commandInputQueue: make(map[string][]mockCommandResponse),
		lookupQueue:       make(map[string][]*queuedLookup),
		calls:             make([]MockCommandCall, 0),
	}
}

func queueKey(command string, args []string) string {
	return command + "\x1f" + strings.Join(args, "\x1f")
}

// EnqueueCommandResult enqueues a deterministic result for RunCommand.
// The response is matched by exact command and argument list and consumed FIFO.
func (m *MockSystem) EnqueueCommandResult(command string, args []string, stdout string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := queueKey(command, args)
	m.commandQueue[key] = append(m.commandQueue[key], mockCommandResponse{stdout: stdout, err: err})
}

// EnqueueCommandWithInputResult enqueues a deterministic result for RunCommandWithInput.
// The response is matched by exact command and argument list and consumed FIFO.
func (m *MockSystem) EnqueueCommandWithInputResult(command string, args []string, stdout string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := queueKey(command, args)
	m.commandInputQueue[key] = append(m.commandInputQueue[key], mockCommandResponse{stdout: stdout, err: err})
}

// EnqueueLookupResult enqueues a deterministic result for Lookup(username).
// The response is consumed FIFO before falling back to in-memory state.
func (m *MockSystem) EnqueueLookupResult(username string, osUserResult *user.User, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var osUserCopy *user.User
	if osUserResult != nil {
		copyValue := *osUserResult
		osUserCopy = &copyValue
	}

	m.lookupQueue[username] = append(m.lookupQueue[username], &queuedLookup{
		osUser: osUserCopy,
		err:    err,
	})
}

// Calls returns a snapshot of command calls made through this mock.
func (m *MockSystem) Calls() []MockCommandCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]MockCommandCall, len(m.calls))
	copy(out, m.calls)
	return out
}

func popQueuedResponse(queue map[string][]mockCommandResponse, key string) (mockCommandResponse, bool) {
	responses, ok := queue[key]
	if !ok || len(responses) == 0 {
		return mockCommandResponse{}, false
	}
	first := responses[0]
	if len(responses) == 1 {
		delete(queue, key)
	} else {
		queue[key] = responses[1:]
	}
	return first, true
}

// AddUser creates or replaces a user in the mock state as both Unix and Samba
// user with an active Samba account.
func (m *MockSystem) AddUser(username, password string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.users[username] = &mockUserState{
		osUser: &user.User{
			Uid:      fmt.Sprintf("%d", m.nextUID),
			Gid:      fmt.Sprintf("%d", m.nextUID),
			Username: username,
			Name:     username,
			HomeDir:  "/home/" + username,
		},
		accountFlags: "U",
		ntHash:       strings.ToUpper(osutil.NTHash(password)),
		hasSamba:     true,
	}
	m.nextUID++
}

// ModifyUser renames a user and updates Samba password if provided.
func (m *MockSystem) ModifyUser(oldUsername, newUsername, password string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.users[oldUsername]
	if !ok {
		return
	}

	delete(m.users, oldUsername)
	state.osUser.Username = newUsername
	if state.osUser.HomeDir == "/home/"+oldUsername {
		state.osUser.HomeDir = "/home/" + newUsername
	}
	if password != "" {
		state.ntHash = strings.ToUpper(osutil.NTHash(password))
	}
	m.users[newUsername] = state
}

// DeleteUser removes a user from both Unix and Samba mock state.
func (m *MockSystem) DeleteUser(username string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, username)
}

// SetSambaAccountFlags updates Samba account flags for an existing user.
func (m *MockSystem) SetSambaAccountFlags(username, flags string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if state, ok := m.users[username]; ok {
		state.accountFlags = flags
	}
}

// SetSambaNTHash updates Samba NT hash for an existing user.
func (m *MockSystem) SetSambaNTHash(username, ntHash string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if state, ok := m.users[username]; ok {
		state.ntHash = strings.ToUpper(ntHash)
	}
}

func (m *MockSystem) commandError(command string, stderr string) error {
	return errors.WithDetails(
		errors.New(command+" failed"),
		"desc", "command execution failed",
		"command", command,
		"stderr", stderr,
	)
}

func (m *MockSystem) commandInputError(command string, stderr string) error {
	return errors.WithDetails(
		errors.New(command+" failed"),
		"desc", "command execution with input failed",
		"command", command,
		"stderr", stderr,
	)
}

func (m *MockSystem) readPassword(stdinContent string) string {
	lines := strings.Split(strings.TrimSpace(stdinContent), "\n")
	if len(lines) == 0 {
		return ""
	}
	return lines[0]
}

// RunCommand simulates command execution against in-memory state.
func (m *MockSystem) RunCommand(ctx context.Context, command string, args ...string) (string, error) {
	_ = ctx

	m.mu.Lock()
	defer m.mu.Unlock()

	argsCopy := append([]string(nil), args...)
	m.calls = append(m.calls, MockCommandCall{
		WithInput: false,
		Command:   command,
		Args:      argsCopy,
	})

	if queued, ok := popQueuedResponse(m.commandQueue, queueKey(command, args)); ok {
		return queued.stdout, queued.err
	}

	switch command {
	case "pdbedit":
		if len(args) >= 4 && args[0] == "-L" && args[1] == "-v" && args[2] == "-u" {
			username := args[3]
			state, ok := m.users[username]
			if !ok || !state.hasSamba {
				return "", m.commandError(command, "Username not found: "+username)
			}
			return fmt.Sprintf("Unix username:        %s\nAccount Flags:        [%-11s]\n", username, state.accountFlags), nil
		}

		if len(args) >= 4 && args[0] == "-L" && args[1] == "-w" && args[2] == "-u" {
			username := args[3]
			state, ok := m.users[username]
			if !ok || !state.hasSamba {
				return "", m.commandError(command, "Username not found: "+username)
			}
			return fmt.Sprintf("%s:%s:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:%s:[%-11s]:LCT-69AD54AA:\n",
				username,
				state.osUser.Uid,
				state.ntHash,
				state.accountFlags,
			), nil
		}

		if len(args) == 1 && args[0] == "-L" {
			keys := make([]string, 0, len(m.users))
			for username, state := range m.users {
				if state.hasSamba {
					keys = append(keys, username)
				}
			}
			slices.Sort(keys)
			var b strings.Builder
			for _, username := range keys {
				state := m.users[username]
				b.WriteString(fmt.Sprintf("%s:%s:%s\n", username, state.osUser.Uid, state.osUser.Name))
			}
			return b.String(), nil
		}

		return "", m.commandError(command, "unsupported pdbedit args")

	case "useradd":
		if len(args) == 0 {
			return "", m.commandError(command, "missing username")
		}
		username := args[len(args)-1]
		if _, exists := m.users[username]; exists {
			return "", m.commandError(command, "useradd: user '"+username+"' already exists")
		}
		m.users[username] = &mockUserState{
			osUser: &user.User{
				Uid:      fmt.Sprintf("%d", m.nextUID),
				Gid:      fmt.Sprintf("%d", m.nextUID),
				Username: username,
				Name:     username,
				HomeDir:  "/home/" + username,
			},
			accountFlags: "U",
			hasSamba:     false,
		}
		m.nextUID++
		return "", nil

	case "smbpasswd":
		if len(args) >= 2 && args[0] == "-x" {
			username := args[1]
			state, ok := m.users[username]
			if !ok || !state.hasSamba {
				return "", m.commandError(command, "Failed to find entry for user "+username)
			}
			state.hasSamba = false
			return "", nil
		}
		return "", m.commandError(command, "unsupported smbpasswd args")

	case "deluser":
		if len(args) == 0 {
			return "", m.commandError(command, "missing username")
		}
		username := args[len(args)-1]
		if _, ok := m.users[username]; !ok {
			return "", m.commandError(command, "user not found")
		}
		delete(m.users, username)
		return "", nil

	case "usermod":
		if len(args) >= 3 && args[0] == "-l" {
			newUsername := args[1]
			oldUsername := args[2]
			if _, exists := m.users[newUsername]; exists {
				return "", m.commandError(command, "new username already exists")
			}
			state, ok := m.users[oldUsername]
			if !ok {
				return "", m.commandError(command, "old username not found")
			}
			delete(m.users, oldUsername)
			state.osUser.Username = newUsername
			m.users[newUsername] = state
			return "", nil
		}

		if len(args) >= 4 && args[0] == "-d" && args[2] == "-m" {
			homeDir := args[1]
			username := args[3]
			state, ok := m.users[username]
			if !ok {
				return "", m.commandError(command, "user not found")
			}
			state.osUser.HomeDir = homeDir
			return "", nil
		}

		return "", m.commandError(command, "unsupported usermod args")
	default:
		return "", m.commandError(command, "unsupported command")
	}
}

// RunCommandWithInput simulates command execution with stdin against in-memory state.
func (m *MockSystem) RunCommandWithInput(ctx context.Context, stdinContent string, command string, args ...string) (string, error) {
	_ = ctx

	m.mu.Lock()
	defer m.mu.Unlock()

	argsCopy := append([]string(nil), args...)
	m.calls = append(m.calls, MockCommandCall{
		WithInput: true,
		Command:   command,
		Args:      argsCopy,
		Stdin:     stdinContent,
	})

	if queued, ok := popQueuedResponse(m.commandInputQueue, queueKey(command, args)); ok {
		return queued.stdout, queued.err
	}

	if command != "smbpasswd" {
		return "", m.commandInputError(command, "unsupported command")
	}

	password := m.readPassword(stdinContent)
	if password == "" {
		return "", m.commandInputError(command, "missing password input")
	}

	if len(args) >= 3 && args[0] == "-a" && args[1] == "-s" {
		username := args[2]
		state, ok := m.users[username]
		if !ok {
			return "", m.commandInputError(command, "system user not found")
		}
		state.hasSamba = true
		state.accountFlags = "U"
		state.ntHash = strings.ToUpper(osutil.NTHash(password))
		return "", nil
	}

	if len(args) >= 2 && args[0] == "-s" {
		username := args[1]
		state, ok := m.users[username]
		if !ok || !state.hasSamba {
			return "", m.commandInputError(command, "samba user not found")
		}
		state.ntHash = strings.ToUpper(osutil.NTHash(password))
		return "", nil
	}

	return "", m.commandInputError(command, "unsupported smbpasswd args")
}

// Lookup returns a mock OS user if present.
func (m *MockSystem) Lookup(username string) (*user.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if queued, ok := m.lookupQueue[username]; ok && len(queued) > 0 {
		first := queued[0]
		if len(queued) == 1 {
			delete(m.lookupQueue, username)
		} else {
			m.lookupQueue[username] = queued[1:]
		}
		if first.osUser != nil {
			copyUser := *first.osUser
			return &copyUser, first.err
		}
		return nil, first.err
	}

	state, ok := m.users[username]
	if !ok || state.osUser == nil {
		return nil, user.UnknownUserError(username)
	}
	copyUser := *state.osUser
	return &copyUser, nil
}
