package converter

import (
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/unixsamba"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserValidityKey(t *testing.T) {
	assert.Equal(t, userValidityKey("admin", "secret"), userValidityKey("admin", "secret"))
	assert.NotEqual(t, userValidityKey("admin", "secret"), userValidityKey("admin", "other"))
	assert.NotEqual(t, userValidityKey("admin", "secret"), userValidityKey("root", "secret"))
	assert.NotContains(t, userValidityKey("admin", "secret"), "secret")
}

func TestCheckUserValidEmptyCredentials(t *testing.T) {
	userValidityCache.Flush()
	t.Cleanup(userValidityCache.Flush)

	assert.False(t, checkUserValid(dbom.SambaUser{}))
	assert.False(t, checkUserValid(dbom.SambaUser{Username: "admin"}))
	assert.False(t, checkUserValid(dbom.SambaUser{Password: "secret"}))
	// Empty credentials short-circuit before any probe, so nothing is cached.
	assert.Equal(t, 0, userValidityCache.ItemCount())
}

func TestCheckUserValidCachedResult(t *testing.T) {
	userValidityCache.Flush()
	t.Cleanup(userValidityCache.Flush)

	user := dbom.SambaUser{Username: "cached-user", Password: "cached-secret"}
	userValidityCache.SetDefault(userValidityKey(user.Username, user.Password), true)

	// True from cache: a live pdbedit probe for this user would fail, so a
	// true result proves no subprocess was spawned.
	assert.True(t, checkUserValid(user))

	userValidityCache.SetDefault(userValidityKey(user.Username, user.Password), false)
	assert.False(t, checkUserValid(user))
}

func TestCheckUserValidStoredAdminDisableOverridesCache(t *testing.T) {
	userValidityCache.Flush()
	t.Cleanup(userValidityCache.Flush)

	user := dbom.SambaUser{Username: "stored-user", Password: "stored-secret", IsValid: new(false)}
	userValidityCache.SetDefault(userValidityKey(user.Username, user.Password), true)

	result := checkUserValidStored(user)
	require.NotNil(t, result)
	assert.False(t, *result)
}

func TestCheckUserValidStoredUsesCachedComputedValidity(t *testing.T) {
	userValidityCache.Flush()
	t.Cleanup(userValidityCache.Flush)

	user := dbom.SambaUser{Username: "computed-user", Password: "computed-secret"}
	userValidityCache.SetDefault(userValidityKey(user.Username, user.Password), true)

	result := checkUserValidStored(user)
	require.NotNil(t, result)
	assert.True(t, *result)
}

func TestCheckUserValidLiveProbeSuccess(t *testing.T) {
	userValidityCache.Flush()
	t.Cleanup(userValidityCache.Flush)

	mock := unixsamba.NewMockSystem()
	mock.AddUser("live-user", "live-secret")
	unixsamba.SetCommandExecutor(mock)
	t.Cleanup(unixsamba.ResetExecutorsToDefaults)

	user := dbom.SambaUser{Username: "live-user", Password: "live-secret"}
	assert.True(t, checkUserValid(user))
	assert.True(t, checkUserValid(user))
	assert.True(t, checkUserValid(user))

	// Three checks for the same credentials must issue a single pdbedit pair
	// (-L -v plus -L -w); the rest is served from cache.
	pdbeditCalls := 0
	for _, call := range mock.Calls() {
		if call.Command == "pdbedit" {
			pdbeditCalls++
		}
	}
	assert.Equal(t, 2, pdbeditCalls)
}

func TestCheckUserValidLiveProbeWrongPassword(t *testing.T) {
	userValidityCache.Flush()
	t.Cleanup(userValidityCache.Flush)

	mock := unixsamba.NewMockSystem()
	mock.AddUser("live-user", "live-secret")
	unixsamba.SetCommandExecutor(mock)
	t.Cleanup(unixsamba.ResetExecutorsToDefaults)

	assert.False(t, checkUserValid(dbom.SambaUser{Username: "live-user", Password: "wrong"}))
}

func TestCheckUserValidCorruptCacheEntryFallsBackToProbe(t *testing.T) {
	userValidityCache.Flush()
	t.Cleanup(userValidityCache.Flush)

	mock := unixsamba.NewMockSystem()
	mock.AddUser("live-user", "live-secret")
	unixsamba.SetCommandExecutor(mock)
	t.Cleanup(unixsamba.ResetExecutorsToDefaults)

	user := dbom.SambaUser{Username: "live-user", Password: "live-secret"}
	userValidityCache.SetDefault(userValidityKey(user.Username, user.Password), "not-a-bool")

	assert.True(t, checkUserValid(user))
}
