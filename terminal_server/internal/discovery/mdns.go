// Package discovery provides service advertisement and LAN discovery support.
package discovery

import "context"

// ServiceInfo identifies the server in LAN discovery responses.
type ServiceInfo struct {
	ServiceType string
	Name        string
	Port        int
	Version     string
}

// Advertiser exposes lifecycle hooks for mDNS service advertisement.
type Advertiser interface {
	Start(ctx context.Context, svc ServiceInfo) error
	Stop(ctx context.Context) error
}

// NoopAdvertiser allows startup flows without mDNS implementation yet.
type NoopAdvertiser struct{}

// Start is a no-op placeholder.
func (NoopAdvertiser) Start(context.Context, ServiceInfo) error { return nil }

// Stop is a no-op placeholder.
func (NoopAdvertiser) Stop(context.Context) error { return nil }
