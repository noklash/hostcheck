package memory

import "testing"

func TestAvailablePercent(t *testing.T) {
	tests := []struct {
		name      string
		total     uint64
		available uint64
		want      float64
	}{
		{
			name:      "half available",
			total:     1000,
			available: 500,
			want:      50,
		},
		{
			name:      "none available",
			total:     1000,
			available: 0,
			want:      0,
		},
		{
			name:      "all available",
			total:     1000,
			available: 1000,
			want:      100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AvailablePercent(MemInfo{
				Total:     tt.total,
				Available: tt.available,
			})
			if err != nil {
				t.Fatalf("AvailablePercent() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("AvailablePercent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAvailablePercentRejectsZeroTotal(t *testing.T) {
	_, err := AvailablePercent(MemInfo{
		Total:     0,
		Available: 0,
	})
	if err == nil {
		t.Fatal("AvailablePercent() expected error for zero total")
	}
}

func TestAvailablePercentRejectsAvailableAboveTotal(t *testing.T) {
	_, err := AvailablePercent(MemInfo{
		Total:     1000,
		Available: 1001,
	})
	if err == nil {
		t.Fatal("AvailablePercent() expected error when available exceeds total")
	}
}
