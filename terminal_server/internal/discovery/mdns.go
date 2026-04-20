// Package discovery contains LAN discovery adapters and service metadata.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/mdns"
)

// ServiceInfo identifies the server in LAN discovery responses.
type ServiceInfo struct {
	ServiceType string
	Name        string
	Port        int
	Version     string
	GRPC        string
	WebSocket   string
	TCP         string
	HTTP        string
	MCP         string
	Priority    []string
}

// Advertiser exposes lifecycle hooks for mDNS service advertisement.
type Advertiser interface {
	Start(ctx context.Context, svc ServiceInfo) error
	Stop(ctx context.Context) error
}

var (
	// ErrMissingServiceType is returned when a service advertisement omits the type.
	ErrMissingServiceType = errors.New("missing service type")
	// ErrMissingServiceName is returned when a service advertisement omits the instance name.
	ErrMissingServiceName = errors.New("missing service name")
	// ErrInvalidPort is returned when a service advertisement has an invalid port.
	ErrInvalidPort = errors.New("invalid service port")
)

// MDNSAdvertiser advertises this server on the local network using mDNS.
type MDNSAdvertiser struct {
	mu        sync.Mutex
	server    *mdns.Server
	newZone   func(ServiceInfo) (*mdns.MDNSService, error)
	newServer func(*mdns.Config) (*mdns.Server, error)
	shutdown  func(*mdns.Server) error
}

// NewMDNSAdvertiser constructs a real mDNS advertiser.
func NewMDNSAdvertiser() *MDNSAdvertiser {
	return &MDNSAdvertiser{
		newZone: func(svc ServiceInfo) (*mdns.MDNSService, error) {
			txt := []string{fmt.Sprintf("version=%s", svc.Version), fmt.Sprintf("name=%s", svc.Name)}
			if grpc := strings.TrimSpace(svc.GRPC); grpc != "" {
				txt = append(txt, fmt.Sprintf("grpc=%s", grpc))
			}
			if ws := strings.TrimSpace(svc.WebSocket); ws != "" {
				txt = append(txt, fmt.Sprintf("ws=%s", ws))
			}
			if tcp := strings.TrimSpace(svc.TCP); tcp != "" {
				txt = append(txt, fmt.Sprintf("tcp=%s", tcp))
			}
			if httpAddr := strings.TrimSpace(svc.HTTP); httpAddr != "" {
				txt = append(txt, fmt.Sprintf("http=%s", httpAddr))
			}
			if mcpAddr := strings.TrimSpace(svc.MCP); mcpAddr != "" {
				txt = append(txt, fmt.Sprintf("mcp=%s", mcpAddr))
			}
			if len(svc.Priority) > 0 {
				txt = append(txt, fmt.Sprintf("priority=%s", strings.Join(svc.Priority, ",")))
			}
			return mdns.NewMDNSService(
				svc.Name,
				svc.ServiceType,
				"",
				"",
				svc.Port,
				nil,
				txt,
			)
		},
		newServer: mdns.NewServer,
		shutdown: func(server *mdns.Server) error {
			return server.Shutdown()
		},
	}
}

// Start begins advertisement. Repeated starts are treated as a no-op.
func (a *MDNSAdvertiser) Start(_ context.Context, svc ServiceInfo) error {
	if err := validateServiceInfo(svc); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.server != nil {
		return nil
	}

	zone, err := a.newZone(svc)
	if err != nil {
		return fmt.Errorf("create mdns zone: %w", err)
	}
	server, err := a.newServer(&mdns.Config{Zone: zone})
	if err != nil {
		return fmt.Errorf("start mdns server: %w", err)
	}
	a.server = server
	return nil
}

// Stop terminates advertisement. Repeated stops are treated as a no-op.
func (a *MDNSAdvertiser) Stop(_ context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.server == nil {
		return nil
	}
	if err := a.shutdown(a.server); err != nil {
		return fmt.Errorf("shutdown mdns server: %w", err)
	}
	a.server = nil
	return nil
}

// NoopAdvertiser allows startup flows without mDNS implementation yet.
type NoopAdvertiser struct{}

// Start is a no-op placeholder.
func (NoopAdvertiser) Start(context.Context, ServiceInfo) error { return nil }

// Stop is a no-op placeholder.
func (NoopAdvertiser) Stop(context.Context) error { return nil }

func validateServiceInfo(svc ServiceInfo) error {
	if svc.ServiceType == "" {
		return ErrMissingServiceType
	}
	if svc.Name == "" {
		return ErrMissingServiceName
	}
	if svc.Port <= 0 || svc.Port > 65535 {
		return ErrInvalidPort
	}
	return nil
}
