package service

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strings"
	"sync"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/grandcat/zeroconf"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// MDNSServiceInterface is the public contract for the mDNS registration service.
// It is called by the WebSocket broker when the Home Assistant custom component
// connects or disconnects via the HELO handshake.
type MDNSServiceInterface interface {
	// OnComponentConnected is called when the HA custom component sends a valid
	// HELO message. It broadcasts the current mDNS registration state so the
	// component can register or unregister the Samba service in Zeroconf.
	OnComponentConnected(message dto.HeloMessage)
	// OnComponentDisconnected is called when the WebSocket connection to the HA
	// custom component is closed. Per the chosen timeout semantics (Option B),
	// no disable message is queued — the component will re-sync on the next
	// HELO when it reconnects.
	OnComponentDisconnected()
}

// mdnsServiceParams groups all dependencies required by MDNSService via fx.In.
type mdnsServiceParams struct {
	fx.In
	Ctx              context.Context
	Broadcaster      BroadcasterServiceInterface
	SettingService   SettingServiceInterface
	EventBus         events.EventBusInterface
	ZeroconfRegister ZeroconfRegister `optional:"true"`
}

// MDNSService broadcasts MdnsRegisterNotification events over WebSocket so the
// Home Assistant custom component can register or unregister the Samba server
// via Zeroconf / mDNS. When the experimental addon-side direct mDNS feature is
// enabled, it also registers the service directly using zeroconf.
type MDNSService struct {
	ctx            context.Context
	broadcaster    BroadcasterServiceInterface
	settingService SettingServiceInterface
	eventBus       events.EventBusInterface

	// connected tracks whether a HA component is currently connected.
	// Access is protected by mu.
	connected bool
	mu        sync.RWMutex

	// zeroconfServer holds the active direct mDNS registration server.
	// Access is protected by mu.
	zeroconfServer ZeroconfServer

	// zeroconfRegister abstracts zeroconf.Register for testability.
	zeroconfRegister ZeroconfRegister
}

// ZeroconfServer is the minimal surface required from a running zeroconf
// registration so tests can substitute a fake implementation.
type ZeroconfServer interface {
	Shutdown()
}

// ZeroconfRegister abstracts direct mDNS registration for testability.
type ZeroconfRegister interface {
	Register(instance, service, domain string, port int, text []string, ifaces []net.Interface) (ZeroconfServer, error)
}

type realZeroconfRegister struct{}

func (realZeroconfRegister) Register(instance, service, domain string, port int, text []string, ifaces []net.Interface) (ZeroconfServer, error) {
	server, err := zeroconf.Register(instance, service, domain, port, text, ifaces)
	if err != nil {
		return nil, err
	}
	return server, nil
}

const (
	mdnsServiceType = "_smb._tcp"
	mdnsDomain      = "local."
	mdnsPort        = 445
)

var nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9]`)

// NewMDNSService constructs the MDNSService and wires lifecycle / event hooks.
func NewMDNSService(lc fx.Lifecycle, params mdnsServiceParams) MDNSServiceInterface {
	reg := params.ZeroconfRegister
	if reg == nil {
		reg = realZeroconfRegister{}
	}
	svc := &MDNSService{
		ctx:              params.Ctx,
		broadcaster:      params.Broadcaster,
		settingService:   params.SettingService,
		eventBus:         params.EventBus,
		zeroconfRegister: reg,
	}

	unsubscribes := svc.setupEventListeners()

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			for _, unsub := range unsubscribes {
				unsub()
			}
			svc.shutdownAddonMDNS()
			return nil
		},
	})

	return svc
}

// setupEventListeners subscribes to server-process and setting events so that
// mDNS state is kept in sync with settings changes and process lifecycle.
func (svc *MDNSService) setupEventListeners() []func() {
	ret := make([]func(), 2)
	ret[0] = svc.eventBus.OnServerProccess(func(ctx context.Context, event events.ServerProcessEvent) errors.E {
		if event.Type == events.EventTypes.CLEAN {
			svc.mu.RLock()
			connected := svc.connected
			svc.mu.RUnlock()
			slog.InfoContext(ctx, "mdns_service: CLEAN event received",
				"connected", connected,
				"broadcasting", connected)
			if connected {
				svc.broadcast(ctx)
			}
		}
		return nil
	})
	ret[1] = svc.eventBus.OnSetting(func(ctx context.Context, event events.SettingEvent) errors.E {
		slog.InfoContext(ctx, "mdns_service: settings changed, reconfiguring direct mDNS")
		svc.reconfigureAddonMDNS(ctx)
		return nil
	})
	return ret
}

// OnComponentConnected is called by the WebSocket broker after a successful HELO
// handshake. It immediately broadcasts the current mDNS registration state.
func (svc *MDNSService) OnComponentConnected(message dto.HeloMessage) {
	svc.mu.Lock()
	svc.connected = true
	svc.mu.Unlock()
	svc.broadcast(svc.ctx)
}

// OnComponentDisconnected is called when the WebSocket connection drops.
// Per Option B timeout semantics, no disable message is sent — the component
// will re-sync when it reconnects.
func (svc *MDNSService) OnComponentDisconnected() {
	svc.mu.Lock()
	svc.connected = false
	svc.mu.Unlock()
}

// broadcast reads the current settings and emits a MdnsRegisterNotification.
func (svc *MDNSService) broadcast(ctx context.Context) {
	settings, err := svc.settingService.Load()
	if err != nil {
		slog.ErrorContext(ctx, "mdns_service: failed to load settings", "err", err)
		return
	}

	enabled := false
	if settings.MDNSRegistration != nil {
		enabled = *settings.MDNSRegistration
	}

	notification := dto.MdnsRegisterNotification{
		Hostname: settings.Hostname,
		Port:     445, // Standard SMB/CIFS port
		Enabled:  enabled,
	}

	slog.InfoContext(ctx, "mdns_service: broadcasting registration notification",
		"enabled", enabled, "hostname", settings.Hostname)

	result := svc.broadcaster.BroadcastGuaranteedMessage(notification)
	slog.InfoContext(ctx, "mdns_service: broadcast result",
		"type", fmt.Sprintf("%T", result), "returned_nil", result == nil)
}

// reconfigureAddonMDNS starts or stops the addon-side direct mDNS registration
// based on the current settings.
func (svc *MDNSService) reconfigureAddonMDNS(ctx context.Context) {
	settings, err := svc.settingService.Load()
	if err != nil {
		slog.ErrorContext(ctx, "mdns_service: failed to load settings for direct mDNS", "err", err)
		return
	}

	enabled := settings.AddonMDNSRegistration != nil && *settings.AddonMDNSRegistration && settings.ExperimentalLabMode
	if !enabled {
		svc.shutdownAddonMDNS()
		return
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	// Tear down any existing registration before re-registering so that changes
	// to hostname or interface selection take effect immediately.
	if svc.zeroconfServer != nil {
		slog.InfoContext(ctx, "mdns_service: shutting down existing direct mDNS registration")
		svc.zeroconfServer.Shutdown()
		svc.zeroconfServer = nil
	}

	instance := sanitizeNetBIOSName(settings.Hostname)
	ifaces, ifaceErr := selectMDNSInterfaces(settings.AddonMDNSInterfaces)
	if ifaceErr != nil {
		slog.ErrorContext(ctx, "mdns_service: failed to select mDNS interfaces", "err", ifaceErr)
		return
	}

	slog.InfoContext(ctx, "mdns_service: registering addon-side direct mDNS",
		"instance", instance,
		"service", mdnsServiceType,
		"port", mdnsPort,
		"ifaces", interfaceNames(ifaces))

	server, regErr := svc.zeroconfRegister.Register(
		instance,
		mdnsServiceType,
		mdnsDomain,
		mdnsPort,
		[]string{"path=/"},
		ifaces,
	)
	if regErr != nil {
		slog.ErrorContext(ctx, "mdns_service: direct mDNS registration failed", "err", regErr)
		return
	}
	svc.zeroconfServer = server
}

// shutdownAddonMDNS stops the addon-side direct mDNS registration.
func (svc *MDNSService) shutdownAddonMDNS() {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if svc.zeroconfServer != nil {
		svc.zeroconfServer.Shutdown()
		svc.zeroconfServer = nil
	}
}

// sanitizeNetBIOSName converts a hostname into a NetBIOS-compatible mDNS
// instance name: uppercase, truncate to 15 characters, replace any
// non-alphanumeric character with '-'.
func sanitizeNetBIOSName(hostname string) string {
	s := strings.ToUpper(hostname)
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	if len(s) > 15 {
		s = s[:15]
	}
	return s
}

// selectMDNSInterfaces returns the network interfaces that should be used for
// addon-side direct mDNS registration. If whitelist is non-empty, only eligible
// interfaces whose name appears in the whitelist are returned. Loopback,
// down, and container/virtual interfaces (docker*, veth*, hassio*, br-*) are
// excluded.
func selectMDNSInterfaces(whitelist []string) ([]net.Interface, error) {
	all, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	wl := make(map[string]struct{}, len(whitelist))
	for _, name := range whitelist {
		wl[name] = struct{}{}
	}
	useWhitelist := len(wl) > 0

	var filtered []net.Interface
	for _, iface := range all {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		name := iface.Name
		if useWhitelist {
			if _, ok := wl[name]; !ok {
				continue
			}
		}
		if isExcludedMDNSInterface(name) {
			continue
		}
		filtered = append(filtered, iface)
	}
	return filtered, nil
}

// isExcludedMDNSInterface returns true for interface names that should never be
// used for mDNS (container/virtual bridges and the loopback interface).
func isExcludedMDNSInterface(name string) bool {
	excludedPrefixes := []string{"lo", "docker", "veth", "hassio", "br-"}
	for _, prefix := range excludedPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// interfaceNames returns a slice of interface names for logging.
func interfaceNames(ifaces []net.Interface) []string {
	names := make([]string, len(ifaces))
	for i, iface := range ifaces {
		names[i] = iface.Name
	}
	return names
}
