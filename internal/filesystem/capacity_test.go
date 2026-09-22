package filesystem

import (
	"testing"
)

func TestUsedBlocks(t *testing.T) {
	stats := Stats{
		BlocksTotal: 1000,
		BlocksFree:  400,
	}

	got, err := stats.UsedBlocks()
	if err != nil {
		t.Fatalf("UsedBlocks() error = %v", err)
	}

	if got != 600 {
		t.Fatalf("UsedBlocks() = %d, want 600", got)
	}
}

func TestUsedBlocksRejectsInvalidState(t *testing.T) {
	stats := Stats{
		BlocksTotal: 100,
		BlocksFree:  101,
	}

	if _, err := stats.UsedBlocks(); err == nil {
		t.Fatal("UsedBlocks() error = nil, want error")
	}
}

func TestAvailableBytes(t *testing.T) {
	stats := Stats{
		BlockSize:       4096,
		BlocksAvailable: 1000,
	}

	got, err := stats.AvailableBytes()
	if err != nil {
		t.Fatalf("AvailableBytes() error = %v", err)
	}

	if got != 4_096_000 {
		t.Fatalf("AvailableBytes() = %d, want 4096000", got)
	}
}

func TestAvailableBytesRejectsZeroBlockSize(t *testing.T) {
	stats := Stats{
		BlockSize:       0,
		BlocksAvailable: 1000,
	}

	if _, err := stats.AvailableBytes(); err == nil {
		t.Fatal("AvailableBytes() error = nil, want error")
	}
}

func TestUsedBytes(t *testing.T) {
	stats := Stats{
		BlockSize:   4096,
		BlocksTotal: 1000,
		BlocksFree:  250,
	}

	got, err := stats.UsedBytes()
	if err != nil {
		t.Fatalf("UsedBytes() error = %v", err)
	}

	if got != 3_072_000 {
		t.Fatalf("UsedBytes() = %d, want 3072000", got)
	}
}

func TestAvailablePercent(t *testing.T) {
	stats := Stats{
		BlocksTotal:     1000,
		BlocksAvailable: 250,
	}

	got, err := stats.AvailablePercent()
	if err != nil {
		t.Fatalf("AvailablePercent() error = %v", err)
	}

	if got != 25 {
		t.Fatalf("AvailablePercent() = %v, want 25", got)
	}
}

func TestUsedInodes(t *testing.T) {
	stats := Stats{
		InodesTotal: 1000,
		InodesFree:  300,
	}

	got, err := stats.UsedInodes()
	if err != nil {
		t.Fatalf("UsedInodes() error = %v", err)
	}

	if got != 700 {
		t.Fatalf("UsedInodes() = %d, want 700", got)
	}
}

func TestAvailableInodePercent(t *testing.T) {
	stats := Stats{
		InodesTotal: 1000,
		InodesFree:  300,
	}

	got, err := stats.AvailableInodePercent()
	if err != nil {
		t.Fatalf("AvailableInodePercent() error = %v", err)
	}

	if got != 30 {
		t.Fatalf("AvailableInodePercent() = %v, want 30", got)
	}
}
