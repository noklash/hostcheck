package network

import (
	"net"
	"testing"
)

func TestReadAddresses(t *testing.T) {
	addresses, err := ReadAddresses("lo")
	if err != nil {
		t.Fatalf("ReadAddresses(lo): %v", err)
	}

	expected := []Address{
		{
			IP:        net.ParseIP("127.0.0.1"),
			PrefixLen: 8,
		},
		{
			IP:        net.ParseIP("::1"),
			PrefixLen: 128,
		},
	}

	if len(addresses) != len(expected) {
		t.Fatalf("got %d addresses, want %d", len(addresses), len(expected))
	}

	for i, want := range expected {
		got := addresses[i]

		if !got.IP.Equal(want.IP) {
			t.Errorf(
				"address %d IP = %s, want %s",
				i,
				got.IP,
				want.IP,
			)
		}

		if got.PrefixLen != want.PrefixLen {
			t.Errorf(
				"address %d prefix = %d, want %d",
				i,
				got.PrefixLen,
				want.PrefixLen,
			)
		}
	}
}

func TestReadAddressesEnp0s3(t *testing.T) {
	addresses, err := ReadAddresses("enp0s3")
	if err != nil {
		t.Fatalf("ReadAddresses(enp0s3): %v", err)
	}

	expected := []Address{
		{
			IP:        net.ParseIP("10.0.2.15"),
			PrefixLen: 24,
		},
		{
			IP:        net.ParseIP("fd17:625c:f037:2:a00:27ff:fed0:dc4d"),
			PrefixLen: 64,
		},
		{
			IP:        net.ParseIP("fe80::a00:27ff:fed0:dc4d"),
			PrefixLen: 64,
		},
	}

	if len(addresses) != len(expected) {
		t.Fatalf(
			"got %d addresses, want %d",
			len(addresses),
			len(expected),
		)
	}

	for i, want := range expected {
		got := addresses[i]

		if !got.IP.Equal(want.IP) {
			t.Errorf(
				"address %d IP = %s, want %s",
				i,
				got.IP,
				want.IP,
			)
		}

		if got.PrefixLen != want.PrefixLen {
			t.Errorf(
				"address %d prefix = %d, want %d",
				i,
				got.PrefixLen,
				want.PrefixLen,
			)
		}
	}
}
