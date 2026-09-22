package filesystem

import "fmt"

// UsedBlocks returns the number of blocks currently in use.
func (s Stats) UsedBlocks() (uint64, error) {
	if s.BlocksFree > s.BlocksTotal {
		return 0, fmt.Errorf(
			"free blocks %d exceed total blocks %d",
			s.BlocksFree,
			s.BlocksTotal,
		)
	}

	return s.BlocksTotal - s.BlocksFree, nil
}

// AvailableBytes returns the number of bytes available to an
// unprivileged process.
func (s Stats) AvailableBytes() (uint64, error) {
	if s.BlockSize == 0 {
		return 0, fmt.Errorf("block size is zero")
	}

	if s.BlocksAvailable > ^uint64(0)/s.BlockSize {
		return 0, fmt.Errorf("available capacity overflows uint64")
	}

	return s.BlocksAvailable * s.BlockSize, nil
}

// UsedBytes returns the number of bytes represented by used blocks.
func (s Stats) UsedBytes() (uint64, error) {
	usedBlocks, err := s.UsedBlocks()
	if err != nil {
		return 0, err
	}

	if s.BlockSize == 0 {
		return 0, fmt.Errorf("block size is zero")
	}

	if usedBlocks > ^uint64(0)/s.BlockSize {
		return 0, fmt.Errorf("used capacity overflows uint64")
	}

	return usedBlocks * s.BlockSize, nil
}

// AvailablePercent returns the percentage of total blocks available
// to an unprivileged process.
func (s Stats) AvailablePercent() (float64, error) {
	if s.BlocksTotal == 0 {
		return 0, fmt.Errorf("total blocks is zero")
	}

	return float64(s.BlocksAvailable) / float64(s.BlocksTotal) * 100, nil
}

// UsedInodes returns the number of allocated inodes.
func (s Stats) UsedInodes() (uint64, error) {
	if s.InodesFree > s.InodesTotal {
		return 0, fmt.Errorf(
			"free inodes %d exceed total inodes %d",
			s.InodesFree,
			s.InodesTotal,
		)
	}

	return s.InodesTotal - s.InodesFree, nil
}

// AvailableInodePercent returns the percentage of inodes still available.
func (s Stats) AvailableInodePercent() (float64, error) {
	if s.InodesTotal == 0 {
		return 0, fmt.Errorf("total inodes is zero")
	}

	return float64(s.InodesFree) / float64(s.InodesTotal) * 100, nil
}
