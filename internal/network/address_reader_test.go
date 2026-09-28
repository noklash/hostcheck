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
			Scope:     254,
		},
		{
			IP:        net.ParseIP("::1"),
			PrefixLen: 128,
			Scope:     254,
		},
	}

	assertAddresses(t, addresses, expected)
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
			Scope:     0,
		},
		{
			IP:        net.ParseIP("fd17:625c:f037:2:a00:27ff:fed0:dc4d"),
			PrefixLen: 64,
			Scope:     0,
		},
		{
			IP:        net.ParseIP("fe80::a00:27ff:fed0:dc4d"),
			PrefixLen: 64,
			Scope:     253,
		},
	}

	assertAddresses(t, addresses, expected)
}

func assertAddresses(t *testing.T, got, want []Address) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %d addresses, want %d", len(got), len(want))
	}

	for i, expected := range want {
		actual := got[i]

		if !actual.IP.Equal(expected.IP) {
			t.Errorf(
				"address %d IP = %s, want %s",
				i,
				actual.IP,
				expected.IP,
			)
		}

		if actual.PrefixLen != expected.PrefixLen {
			t.Errorf(
				"address %d prefix = %d, want %d",
				i,
				actual.PrefixLen,
				expected.PrefixLen,
			)
		}

		if actual.Scope != expected.Scope {
			t.Errorf(
				"address %d scope = %d, want %d",
				i,
				actual.Scope,
				expected.Scope,
			)
		}
	}
}
