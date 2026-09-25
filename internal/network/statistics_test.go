package network

import (
	"testing"
)

func TestReadStatisticsLoopback(t *testing.T) {
	stats, err := ReadStatistics("lo")
	if err != nil {
		t.Fatalf("ReadStatistics(lo) error = %v", err)
	}

	if stats.RXBytes == 0 {
		t.Fatal("RXBytes = 0, want non-zero")
	}

	if stats.RXPackets == 0 {
		t.Fatal("RXPackets = 0, want non-zero")
	}

	if stats.TXBytes == 0 {
		t.Fatal("TXBytes = 0, want non-zero")
	}

	if stats.TXPackets == 0 {
		t.Fatal("TXPackets = 0, want non-zero")
	}
}

func TestReadStatisticsEthernet(t *testing.T) {
	stats, err := ReadStatistics("enp0s3")
	if err != nil {
		t.Fatalf("ReadStatistics(enp0s3) error = %v", err)
	}

	if stats.RXBytes == 0 {
		t.Fatal("RXBytes = 0, want non-zero")
	}

	if stats.RXPackets == 0 {
		t.Fatal("RXPackets = 0, want non-zero")
	}

	if stats.TXBytes == 0 {
		t.Fatal("TXBytes = 0, want non-zero")
	}

	if stats.TXPackets == 0 {
		t.Fatal("TXPackets = 0, want non-zero")
	}
}
