package discovery

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/mdns"
)

func TestValidateServiceInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		svc     ServiceInfo
		wantErr error
	}{
		{
			name: "valid",
			svc: ServiceInfo{
				ServiceType: "_terminals._tcp.local.",
				Name:        "HomeServer",
				Port:        50051,
			},
		},
		{
			name: "missing service type",
			svc: ServiceInfo{
				Name: "HomeServer",
				Port: 50051,
			},
			wantErr: ErrMissingServiceType,
		},
		{
			name: "missing name",
			svc: ServiceInfo{
				ServiceType: "_terminals._tcp.local.",
				Port:        50051,
			},
			wantErr: ErrMissingServiceName,
		},
		{
			name: "invalid port",
			svc: ServiceInfo{
				ServiceType: "_terminals._tcp.local.",
				Name:        "HomeServer",
				Port:        70000,
			},
			wantErr: ErrInvalidPort,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateServiceInfo(tt.svc)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("validateServiceInfo returned unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validateServiceInfo error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestMDNSAdvertiserStartStopIdempotent(t *testing.T) {
	t.Parallel()

	startCalls := 0
	shutdownCalls := 0
	advertiser := &MDNSAdvertiser{
		newZone: func(ServiceInfo) (*mdns.MDNSService, error) {
			return &mdns.MDNSService{}, nil
		},
		newServer: func(*mdns.Config) (*mdns.Server, error) {
			startCalls++
			return &mdns.Server{}, nil
		},
		shutdown: func(*mdns.Server) error {
			shutdownCalls++
			return nil
		},
	}

	svc := ServiceInfo{
		ServiceType: "_terminals._tcp.local.",
		Name:        "HomeServer",
		Port:        50051,
		Version:     "1",
	}

	if err := advertiser.Start(context.Background(), svc); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := advertiser.Start(context.Background(), svc); err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
	if startCalls != 1 {
		t.Fatalf("start calls = %d, want 1", startCalls)
	}

	if err := advertiser.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if err := advertiser.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop() error = %v", err)
	}
	if shutdownCalls != 1 {
		t.Fatalf("shutdown calls = %d, want 1", shutdownCalls)
	}
}

func TestMDNSAdvertiserStartValidation(t *testing.T) {
	t.Parallel()

	advertiser := NewMDNSAdvertiser()
	err := advertiser.Start(context.Background(), ServiceInfo{Name: "HomeServer", Port: 50051})
	if !errors.Is(err, ErrMissingServiceType) {
		t.Fatalf("Start() error = %v, want ErrMissingServiceType", err)
	}
}
