package network

type Statistics struct {
	RXBytes   uint64
	RXPackets uint64
	RXErrors  uint64
	RXDropped uint64
	TXBytes   uint64
	TXPackets uint64
	TXErrors  uint64
	TXDropped uint64
}
