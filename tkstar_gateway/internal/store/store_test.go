package store

import (
	"testing"
	"time"

	"github.com/ArrowSK/ha-tkstar-gateway/internal/model"
)

func TestHistoryAndStats(t *testing.T) {
	st, err := New(t.TempDir(), 180)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	points := []model.Position{
		{ReceivedAt: base, Latitude: 47.5, Longitude: 19.05, Valid: true, SpeedKmh: 10},
		{ReceivedAt: base.Add(5 * time.Minute), Latitude: 47.51, Longitude: 19.06, Valid: true, SpeedKmh: 25},
	}
	for _, point := range points {
		if err := st.AppendPosition("synthetic", point); err != nil {
			t.Fatal(err)
		}
	}
	got, err := st.Positions("synthetic", base.Add(-time.Minute), base.Add(time.Hour), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d points", len(got))
	}
	stats := ComputeStats(got)
	if stats.Points != 2 || stats.MaxSpeedKmh != 25 || stats.DistanceKm <= 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
