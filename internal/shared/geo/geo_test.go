package geo

import "testing"

func TestHaversineKm(t *testing.T) {
	// Jakarta (-6.2, 106.816) to Bandung (-6.9175, 107.6191) ~ 115-120 km
	d := HaversineKm(-6.2, 106.816, -6.9175, 107.6191)
	if d < 100 || d > 140 {
		t.Fatalf("unexpected distance: %v", d)
	}
}
