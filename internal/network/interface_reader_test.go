package network

import (
	"testing"
)

func TestReadInterfaceLoopback(t *testing.T) {
	iface, err := ReadInterface("lo")
	if err != nil {
		t.Fatalf("ReadInterface(lo) error = %v", err)
	}

	if iface.Name != "lo" {
		t.Fatalf("Name = %q, want %q", iface.Name, "lo")
	}

	if iface.OperState != "unknown" {
		t.Fatalf("OperState = %q, want %q", iface.OperState, "unknown")
	}

	if !iface.Carrier {
		t.Fatal("Carrier = false, want true")
	}

	if iface.MTU != 65536 {
		t.Fatalf("MTU = %d, want 65536", iface.MTU)
	}

	if iface.SpeedMbps != nil {
		t.Fatalf("SpeedMbps = %v, want nil", *iface.SpeedMbps)
	}

	if iface.Duplex != nil {
		t.Fatalf("Duplex = %q, want nil", *iface.Duplex)
	}
}

func TestReadInterfaceEthernet(t *testing.T) {
	iface, err := ReadInterface("enp0s3")
	if err != nil {
		t.Fatalf("ReadInterface(enp0s3) error = %v", err)
	}

	if iface.Name != "enp0s3" {
		t.Fatalf("Name = %q, want %q", iface.Name, "enp0s3")
	}

	if iface.OperState != "up" {
		t.Fatalf("OperState = %q, want %q", iface.OperState, "up")
	}

	if !iface.Carrier {
		t.Fatal("Carrier = false, want true")
	}

	if iface.MTU != 1500 {
		t.Fatalf("MTU = %d, want 1500", iface.MTU)
	}

	if iface.SpeedMbps == nil {
		t.Fatal("SpeedMbps = nil, want value")
	}

	if *iface.SpeedMbps != 1000 {
		t.Fatalf("SpeedMbps = %d, want 1000", *iface.SpeedMbps)
	}

	if iface.Duplex == nil {
		t.Fatal("Duplex = nil, want value")
	}

	if *iface.Duplex != "full" {
		t.Fatalf("Duplex = %q, want %q", *iface.Duplex, "full")
	}
}
