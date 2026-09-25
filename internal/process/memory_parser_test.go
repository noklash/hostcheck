package process

import "testing"

func TestParseStatusUserspaceProcess(t *testing.T) {
	input := `Name:	test
Kthread:	0
VmSize:	   118468 kB
VmRSS:	    112340 kB
RssAnon:	  105556 kB
RssFile:	    6784 kB
RssShmem:	       0 kB
`

	got, err := ParseStatus(input)
	if err != nil {
		t.Fatalf("ParseStatus() error = %v", err)
	}

	if got.Kthread {
		t.Fatal("Kthread = true, want false")
	}

	if got.Memory == nil {
		t.Fatal("Memory = nil, want memory data")
	}

	tests := []struct {
		name string
		got  uint64
		want uint64
	}{
		{"virtual", got.Memory.VirtualBytes, 118468 * 1024},
		{"resident", got.Memory.ResidentBytes, 112340 * 1024},
		{"anonymous", got.Memory.AnonymousBytes, 105556 * 1024},
		{"file backed", got.Memory.FileBackedBytes, 6784 * 1024},
		{"shared memory", got.Memory.SharedMemoryBytes, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %d, want %d", tt.got, tt.want)
			}
		})
	}
}

func TestParseStatusKernelThread(t *testing.T) {
	input := `Name:	kworker/0:0H-kblockd
Kthread:	1
Threads:	1
`

	got, err := ParseStatus(input)
	if err != nil {
		t.Fatalf("ParseStatus() error = %v", err)
	}

	if !got.Kthread {
		t.Fatal("Kthread = false, want true")
	}

	if got.Memory != nil {
		t.Fatal("Memory != nil, want nil for kernel thread")
	}
}

func TestParseStatusMissingKthread(t *testing.T) {
	input := `Name:	test
VmSize:	   100 kB
VmRSS:	    80 kB
RssAnon:	    20 kB
RssFile:	    60 kB
RssShmem:	     0 kB
`

	if _, err := ParseStatus(input); err == nil {
		t.Fatal("expected error for missing Kthread")
	}
}

func TestParseStatusInvalidKthread(t *testing.T) {
	input := `Kthread:	2
`

	if _, err := ParseStatus(input); err == nil {
		t.Fatal("expected error for invalid Kthread")
	}
}

func TestParseStatusMissingMemoryField(t *testing.T) {
	input := `Kthread:	0
VmSize:	   100 kB
VmRSS:	    80 kB
RssAnon:	    20 kB
RssFile:	    60 kB
`

	if _, err := ParseStatus(input); err == nil {
		t.Fatal("expected error for missing memory field")
	}
}

func TestParseStatusInvalidMemoryUnit(t *testing.T) {
	input := `Kthread:	0
VmSize:	   100 MB
VmRSS:	    80 kB
RssAnon:	    20 kB
RssFile:	    60 kB
RssShmem:	     0 kB
`

	if _, err := ParseStatus(input); err == nil {
		t.Fatal("expected error for invalid memory unit")
	}
}
