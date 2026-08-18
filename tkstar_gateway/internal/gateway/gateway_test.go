package gateway

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ArrowSK/ha-tkstar-gateway/internal/geolocation"
	"github.com/ArrowSK/ha-tkstar-gateway/internal/model"
	"github.com/ArrowSK/ha-tkstar-gateway/internal/store"
)

func TestPairingAndOverspeed(t *testing.T) {
	root := t.TempDir()
	st, err := store.New(root, 180)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := New(st, geolocation.New(root, ""), log, 10*time.Minute)

	const id = "SYNTHETIC0001"
	if gw.Accept(id, "watch") {
		t.Fatal("unknown tracker must be rejected while pairing is closed")
	}
	gw.StartPairing(time.Minute)
	if !gw.Accept(id, "watch") {
		t.Fatal("tracker should be accepted while pairing is open")
	}

	device := gw.Devices()[0]
	if _, err := gw.Update(device.Key, map[string]any{"overspeed_kmh": float64(50)}); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	gw.Handle(model.Message{
		DeviceID: id,
		Protocol: "watch",
		Position: &model.Position{
			Timestamp:  now,
			ReceivedAt: now,
			Latitude:   47.5,
			Longitude:  19.05,
			Valid:      true,
			Source:     "GPS",
			SpeedKmh:   70,
			Battery:    90,
			GSM:        80,
			Protocol:   "watch",
		},
	})

	alarms, err := st.Alarms(device.Key, now.Add(-time.Minute), now.Add(time.Minute), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(alarms) != 1 || alarms[0].Alarm != "overspeed" {
		t.Fatalf("unexpected alarms: %+v", alarms)
	}
}
