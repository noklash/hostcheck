package process

import "testing"

func TestReadMemoryInvalidPID(t *testing.T) {
	tests := []int64{0, -1}

	for _, pid := range tests {
		t.Run("invalid", func(t *testing.T) {
			if _, err := ReadMemory(pid); err == nil {
				t.Fatalf("ReadMemory(%d) expected error", pid)
			}
		})
	}
}
