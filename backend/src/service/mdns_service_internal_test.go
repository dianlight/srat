// Package service contains white-box unit tests for the unexported mDNS
// helpers and the error paths of MDNSService.
package service

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/server/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
)

func TestSanitizeNetBIOSName(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		want     string
	}{
		{"lowercase is uppercased", "server", "SERVER"},
		{"already uppercase", "SERVER", "SERVER"},
		{"truncated to 15 chars", "this-hostname-is-way-too-long", "THIS-HOSTNAME-I"},
		{"exactly 15 chars kept", "abcdefghijklmno", "ABCDEFGHIJKLMNO"},
		{"non-alphanumeric replaced with dash", "my server@01!", "MY-SERVER-01-"},
		{"dots and underscores replaced", "srv-01.test_foo", "SRV-01-TEST-FOO"},
		{"empty string", "", ""},
		{"non-ASCII characters replaced", "sévre", "S-VRE"},
		{"all special chars", "!!!", "---"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, sanitizeNetBIOSName(tc.hostname))
		})
	}
}

func TestIsExcludedMDNSInterface(t *testing.T) {
	tests := []struct {
		name  string
		iface string
		want  bool
	}{
		{"loopback", "lo", true},
		{"loopback numbered", "lo0", true},
		{"docker bridge", "docker0", true},
		{"veth pair", "veth123abc", true},
		{"hassio bridge", "hassio", true},
		{"docker bridge with dash", "br-1234", true},
		{"plain br interface is not excluded", "br0", false},
		{"ethernet", "en0", false},
		{"wifi", "wlan0", false},
		{"ethernet eth", "eth0", false},
		{"empty", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isExcludedMDNSInterface(tc.iface))
		})
	}
}

func TestInterfaceNames(t *testing.T) {
	t.Run("nil input returns empty slice", func(t *testing.T) {
		assert.Empty(t, interfaceNames(nil))
	})

	t.Run("returns names in order", func(t *testing.T) {
		ifaces := []net.Interface{{Name: "en0"}, {Name: "eth0"}}
		assert.Equal(t, []string{"en0", "eth0"}, interfaceNames(ifaces))
	})
}

// TestSelectMDNSInterfaces verifies the invariants of the interface filter:
// returned interfaces must be up, non-loopback and never excluded container
// bridges. Whitelisting must further restrict the result to the named entry.
func TestSelectMDNSInterfaces(t *testing.T) {
	ifaces, err := selectMDNSInterfaces(nil)
	require.NoError(t, err)
	for _, iface := range ifaces {
		assert.NotZero(t, iface.Flags&net.FlagUp, "returned interface %s must be up", iface.Name)
		assert.Zero(t, iface.Flags&net.FlagLoopback, "returned interface %s must not be loopback", iface.Name)
		assert.False(t, isExcludedMDNSInterface(iface.Name), "returned interface %s must not be excluded", iface.Name)
	}

	if len(ifaces) > 0 {
		wl := []string{ifaces[0].Name}
		filtered, err := selectMDNSInterfaces(wl)
		require.NoError(t, err)
		for _, iface := range filtered {
			assert.Equal(t, wl[0], iface.Name, "whitelist must restrict the returned interfaces")
		}
	}
}

// internalStubSettings is a hand-written SettingServiceInterface fake whose
// Load behavior is configurable for error-path testing.
type internalStubSettings struct {
	settings *dto.Settings
	err      errors.E
}

func (s *internalStubSettings) Load() (*dto.Settings, errors.E) { return s.settings, s.err }
func (s *internalStubSettings) UpdateSettings(*dto.Settings) errors.E {
	return nil
}
func (s *internalStubSettings) SetCommandExists(func(cmd []string) bool) {}
func (s *internalStubSettings) DumpTable() (string, errors.E)            { return "", nil }

// internalStubBroadcaster records BroadcastGuaranteedMessage calls.
type internalStubBroadcaster struct {
	mu         sync.Mutex
	broadcasts int
}

func (b *internalStubBroadcaster) BroadcastMessage(msg any) any { return nil }
func (b *internalStubBroadcaster) BroadcastGuaranteedMessage(msg any) any {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.broadcasts++
	return nil
}
func (b *internalStubBroadcaster) ProcessWebSocketChannel(send ws.Sender) {}

func (b *internalStubBroadcaster) broadcastCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.broadcasts
}

// internalFakeServer records Shutdown calls.
type internalFakeServer struct {
	shutdownCalled bool
}

func (s *internalFakeServer) Shutdown() { s.shutdownCalled = true }

// internalFakeRegister is a configurable ZeroconfRegister fake.
type internalFakeRegister struct {
	mu      sync.Mutex
	calls   int
	err     error
	servers []*internalFakeServer
}

func (r *internalFakeRegister) Register(instance, service, domain string, port int, text []string, ifaces []net.Interface) (ZeroconfServer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	s := &internalFakeServer{}
	r.servers = append(r.servers, s)
	return s, nil
}

func (r *internalFakeRegister) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func (r *internalFakeRegister) lastServer() *internalFakeServer {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.servers) == 0 {
		return nil
	}
	return r.servers[len(r.servers)-1]
}

// directMDNSSettings returns Settings with the direct (non-proxy) mDNS mode
// active: master switch on and use_component_mdns_proxy explicitly false.
func directMDNSSettings(hostname string) *dto.Settings {
	return &dto.Settings{
		Hostname:              hostname,
		MDNSRegistration:      new(true),
		UseComponentMDNSProxy: new(false),
	}
}

// TestBroadcast_LoadError verifies broadcast() returns early without emitting
// when settings cannot be loaded.
func TestBroadcast_LoadError(t *testing.T) {
	broadcaster := &internalStubBroadcaster{}
	svc := &MDNSService{
		ctx:            context.Background(),
		broadcaster:    broadcaster,
		settingService: &internalStubSettings{err: errors.New("load failed")},
	}

	svc.broadcast(context.Background())

	assert.Equal(t, 0, broadcaster.broadcastCount(), "no notification must be broadcast when Load fails")
}

// TestReconfigureDirectMDNS_LoadError verifies reconfigureDirectMDNS returns
// early when settings cannot be loaded.
func TestReconfigureDirectMDNS_LoadError(t *testing.T) {
	register := &internalFakeRegister{}
	svc := &MDNSService{
		ctx:              context.Background(),
		settingService:   &internalStubSettings{err: errors.New("load failed")},
		zeroconfRegister: register,
	}

	svc.reconfigureDirectMDNS(context.Background())

	assert.Equal(t, 0, register.callCount(), "no registration must be attempted when Load fails")
}

// TestReconfigureDirectMDNS_NilSettings verifies the nil-settings guard that
// protects the OnStart hook path where settings may not be available yet.
func TestReconfigureDirectMDNS_NilSettings(t *testing.T) {
	register := &internalFakeRegister{}
	svc := &MDNSService{
		ctx:              context.Background(),
		settingService:   &internalStubSettings{settings: nil},
		zeroconfRegister: register,
	}

	svc.reconfigureDirectMDNS(context.Background())

	assert.Equal(t, 0, register.callCount(), "no registration must be attempted when settings are nil")
}

// TestReconfigureDirectMDNS_RegisterError verifies that a zeroconf registration
// failure is logged and leaves no server behind.
func TestReconfigureDirectMDNS_RegisterError(t *testing.T) {
	register := &internalFakeRegister{err: errors.New("register failed")}
	svc := &MDNSService{
		ctx:              context.Background(),
		settingService:   &internalStubSettings{settings: directMDNSSettings("server")},
		zeroconfRegister: register,
	}

	svc.reconfigureDirectMDNS(context.Background())

	assert.Equal(t, 1, register.callCount(), "registration must be attempted once")
	svc.mu.RLock()
	assert.Nil(t, svc.zeroconfServer, "no server must be retained after a registration error")
	svc.mu.RUnlock()
}

// TestReconfigureDirectMDNS_ShutsDownExistingServer verifies that an active
// registration is torn down before a new one is created on reconfigure.
func TestReconfigureDirectMDNS_ShutsDownExistingServer(t *testing.T) {
	register := &internalFakeRegister{}
	oldServer := &internalFakeServer{}
	svc := &MDNSService{
		ctx:              context.Background(),
		settingService:   &internalStubSettings{settings: directMDNSSettings("server")},
		zeroconfRegister: register,
		zeroconfServer:   oldServer,
	}

	svc.reconfigureDirectMDNS(context.Background())

	require.Equal(t, 1, register.callCount())
	assert.True(t, oldServer.shutdownCalled, "existing registration must be shut down before re-registering")
	svc.mu.RLock()
	assert.Same(t, register.lastServer(), svc.zeroconfServer)
	svc.mu.RUnlock()
}

// TestReconfigureDirectMDNS_DisabledShutsDownServer verifies that switching to
// component-proxy/disabled mode tears down any active direct registration.
func TestReconfigureDirectMDNS_DisabledShutsDownServer(t *testing.T) {
	register := &internalFakeRegister{}
	oldServer := &internalFakeServer{}
	svc := &MDNSService{
		ctx: context.Background(),
		settingService: &internalStubSettings{settings: &dto.Settings{
			Hostname:              "server",
			MDNSRegistration:      new(true),
			UseComponentMDNSProxy: new(true),
		}},
		zeroconfRegister: register,
		zeroconfServer:   oldServer,
	}

	svc.reconfigureDirectMDNS(context.Background())

	assert.Equal(t, 0, register.callCount(), "proxy mode must not register direct mDNS")
	assert.True(t, oldServer.shutdownCalled, "existing direct registration must be shut down")
	svc.mu.RLock()
	assert.Nil(t, svc.zeroconfServer)
	svc.mu.RUnlock()
}
