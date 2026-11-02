package appwrite

import "testing"

func TestSunnahXPDelta(t *testing.T) {
	tests := []struct {
		name     string
		oldCount int
		newCount int
		want     int
	}{
		{"no change", 3, 3, 0},
		{"add three practices", 0, 3, 30},
		{"add two practices", 3, 5, 20},
		{"resubmit same selection", 5, 5, 0},
		{"remove two practices", 5, 3, -20},
		{"reach all ten", 9, 10, 60},
		{"drop below all ten", 10, 9, -60},
		{"zero to all ten", 0, 10, 150},
		{"all ten to zero", 10, 0, -150},
		{"remove one practice", 5, 4, -10},
		{"zero to one", 0, 1, 10},
		{"one to zero", 1, 0, -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sunnahXPDelta(tt.oldCount, tt.newCount); got != tt.want {
				t.Errorf("sunnahXPDelta(%d, %d) = %d, want %d", tt.oldCount, tt.newCount, got, tt.want)
			}
		})
	}
}

func TestSunnahXPDeltaIsIdempotent(t *testing.T) {
	// Re-applying the same selection must never award XP again.
	for count := 0; count <= SunnahTickTotal; count++ {
		if got := sunnahXPDelta(count, count); got != 0 {
			t.Errorf("sunnahXPDelta(%d, %d) = %d, want 0", count, count, got)
		}
	}
}

func TestUniqueCount(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		want  int
	}{
		{"nil", nil, 0},
		{"empty strings ignored", []string{"", ""}, 0},
		{"practices", []string{"Qiamullail", "Ma'thurat"}, 2},
		{"duplicates collapse", []string{"Ma'thurat", "Ma'thurat", "Dhuha Prayer"}, 2},
		{"mixed with empty", []string{"", "Selawat 100×", ""}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uniqueCount(tt.items); got != tt.want {
				t.Errorf("uniqueCount(%v) = %d, want %d", tt.items, got, tt.want)
			}
		})
	}
}
