package network

import (
	"testing"
)

func TestEnumerateInterfaces(t *testing.T) {
	names, err := EnumerateInterfaces()
	if err != nil {
		t.Fatalf("EnumerateInterfaces() error = %v", err)
	}

	if len(names) == 0 {
		t.Fatal("EnumerateInterfaces() returned no interfaces")
	}

	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("interfaces are not sorted: %v", names)
		}
	}
}
