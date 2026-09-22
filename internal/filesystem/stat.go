package filesystem

// Stats contains filesystem capacity and inode statistics.
// Block and inode counts are kept in the units reported by statfs.
type Stats struct {
	Path            string
	BlockSize       uint64
	BlocksTotal     uint64
	BlocksFree      uint64
	BlocksAvailable uint64
	InodesTotal     uint64
	InodesFree      uint64
}
