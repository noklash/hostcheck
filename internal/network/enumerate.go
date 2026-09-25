package network

import (
	"os"
	"path/filepath"
	"sort"
)

const sysClassNet = "/sys/class/net"

func EnumerateInterfaces() ([]string, error) {
	entries, err := os.ReadDir(sysClassNet)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))

	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	sort.Strings(names)

	return names, nil
}

func interfacePath(name string) string {
	return filepath.Join(sysClassNet, name)
}
