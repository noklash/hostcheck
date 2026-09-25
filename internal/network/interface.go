package network

type Interface struct {
	Name         string
	HardwareAddr string
	OperState    string
	Carrier      bool
	MTU          uint64
	SpeedMbps    *uint64
	Duplex       *string
}
