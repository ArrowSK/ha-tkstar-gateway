package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ArrowSK/ha-tkstar-gateway/internal/geolocation"
	"github.com/ArrowSK/ha-tkstar-gateway/internal/model"
	mq "github.com/ArrowSK/ha-tkstar-gateway/internal/mqtt"
	"github.com/ArrowSK/ha-tkstar-gateway/internal/store"
)

type Gateway struct {
	mu            sync.RWMutex
	devices       map[string]*model.Device
	pairingUntil  time.Time
	store         *store.Store
	geo           *geolocation.Resolver
	mqtt          *mq.Client
	log           *slog.Logger
	offlineAfter  time.Duration
	lastOverspeed map[string]time.Time
}

func New(st *store.Store, geo *geolocation.Resolver, log *slog.Logger, offline time.Duration) *Gateway {
	g := &Gateway{
		devices:       map[string]*model.Device{},
		store:         st,
		geo:           geo,
		log:           log,
		offlineAfter:  offline,
		lastOverspeed: map[string]time.Time{},
	}
	ds, _ := st.LoadDevices()
	for i := range ds {
		d := ds[i]
		g.devices[d.ID] = &d
	}
	return g
}

func (g *Gateway) SetMQTT(m *mq.Client) {
	g.mu.Lock()
	g.mqtt = m
	ds := make([]model.Device, 0, len(g.devices))
	for _, d := range g.devices {
		ds = append(ds, *d)
	}
	g.mu.Unlock()
	for _, d := range ds {
		m.Discover(d)
		m.State(d)
	}
}

func hashKey(id string) string {
	h := sha256.Sum256([]byte(id))
	return hex.EncodeToString(h[:])[:12]
}

func friendly(id string) string {
	x := id
	if len(x) > 4 {
		x = x[len(x)-4:]
	}
	return "Tracker " + x
}

func (g *Gateway) Accept(id, proto string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}

	g.mu.Lock()
	if d, ok := g.devices[id]; ok {
		if d.Protocol == "" {
			d.Protocol = proto
		}
		g.mu.Unlock()
		return true
	}
	if time.Now().After(g.pairingUntil) {
		g.mu.Unlock()
		return false
	}

	d := &model.Device{
		ID:        id,
		Key:       hashKey(id),
		Name:      friendly(id),
		Protocol:  proto,
		CreatedAt: time.Now().UTC(),
	}
	g.devices[id] = d
	g.persistLocked()
	m := g.mqtt
	copy := *d
	g.mu.Unlock()

	if m != nil {
		m.Discover(copy)
	}
	g.log.Info("paired new tracker", "key", d.Key, "protocol", proto)
	return true
}

func (g *Gateway) Handle(msg model.Message) {
	g.mu.Lock()
	d := g.devices[msg.DeviceID]
	if d == nil {
		g.mu.Unlock()
		return
	}

	d.LastSeen = time.Now().UTC()
	d.Protocol = msg.Protocol
	if msg.ICCID != "" {
		d.ICCID = msg.ICCID
	}
	if msg.Battery != nil {
		d.Battery = *msg.Battery
	}
	if msg.GSM != nil {
		d.GSM = *msg.GSM
	}

	if d.Privacy {
		g.persistLocked()
		copy := *d
		m := g.mqtt
		g.mu.Unlock()
		if m != nil {
			m.State(copy)
		}
		return
	}

	p := msg.Position
	if p != nil {
		p.DeviceID = ""
		if p.Valid {
			g.geo.Learn(*p)
		} else {
			g.geo.Resolve(p)
		}

		isLBS := strings.HasPrefix(p.Source, "LBS")
		if p.Valid && !(d.FilterLBS && isLBS) {
			if d.OverspeedKmh > 0 && p.SpeedKmh > d.OverspeedKmh {
				last := g.lastOverspeed[d.Key]
				if time.Since(last) >= 5*time.Minute {
					if p.Alarm == "" {
						p.Alarm = "overspeed"
					} else if !strings.Contains(p.Alarm, "overspeed") {
						p.Alarm += ",overspeed"
					}
					g.lastOverspeed[d.Key] = time.Now()
				}
			}

			d.Last = p
			if p.Battery >= 0 {
				d.Battery = p.Battery
			}
			if p.GSM >= 0 {
				d.GSM = p.GSM
			}
			_ = g.store.AppendPosition(d.Key, *p)
			if p.Alarm != "" {
				_ = g.store.AppendAlarm(d.Key, *p)
			}
		}
	}

	g.persistLocked()
	copy := *d
	m := g.mqtt
	g.mu.Unlock()
	if m != nil {
		m.State(copy)
	}
}

func (g *Gateway) persistLocked() {
	ds := make([]model.Device, 0, len(g.devices))
	for _, d := range g.devices {
		ds = append(ds, *d)
	}
	_ = g.store.SaveDevices(ds)
}

func (g *Gateway) StartPairing(d time.Duration) time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pairingUntil = time.Now().Add(d)
	return g.pairingUntil
}

func (g *Gateway) StopPairing() {
	g.mu.Lock()
	g.pairingUntil = time.Time{}
	g.mu.Unlock()
}

func (g *Gateway) PairingUntil() time.Time {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.pairingUntil
}

func (g *Gateway) OfflineAfter() time.Duration {
	return g.offlineAfter
}

func (g *Gateway) Devices() []model.Device {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]model.Device, 0, len(g.devices))
	for _, d := range g.devices {
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (g *Gateway) DeviceByKey(k string) (model.Device, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, d := range g.devices {
		if d.Key == k {
			return *d, true
		}
	}
	return model.Device{}, false
}

func (g *Gateway) Update(k string, p map[string]any) (model.Device, error) {
	g.mu.Lock()
	var d *model.Device
	for _, x := range g.devices {
		if x.Key == k {
			d = x
			break
		}
	}
	if d == nil {
		g.mu.Unlock()
		return model.Device{}, errors.New("not found")
	}

	if v, ok := p["name"].(string); ok && strings.TrimSpace(v) != "" {
		d.Name = strings.TrimSpace(v)
	}
	if v, ok := p["model"].(string); ok {
		d.Model = strings.TrimSpace(v)
	}
	if v, ok := p["sim"].(string); ok {
		d.SIM = strings.TrimSpace(v)
	}
	if v, ok := p["iccid"].(string); ok {
		d.ICCID = strings.TrimSpace(v)
	}
	if v, ok := p["plate"].(string); ok {
		d.Plate = strings.TrimSpace(v)
	}
	if v, ok := p["icon"].(string); ok {
		d.Icon = strings.TrimSpace(v)
	}
	if v, ok := p["privacy"].(bool); ok {
		d.Privacy = v
	}
	if v, ok := p["filter_lbs"].(bool); ok {
		d.FilterLBS = v
	}
	if v, ok := p["overspeed_kmh"].(float64); ok && v >= 0 && v <= 400 {
		d.OverspeedKmh = v
	}

	g.persistLocked()
	copy := *d
	m := g.mqtt
	g.mu.Unlock()
	if m != nil {
		m.Discover(copy)
		m.State(copy)
	}
	return copy, nil
}

func (g *Gateway) SetPrivacyByKey(k string, v bool) {
	_, _ = g.Update(k, map[string]any{"privacy": v})
}

func (g *Gateway) MarkOffline() {
	g.mu.RLock()
	if g.mqtt == nil {
		g.mu.RUnlock()
		return
	}
	now := time.Now()
	var offline []model.Device
	for _, d := range g.devices {
		if !d.LastSeen.IsZero() && now.Sub(d.LastSeen) > g.offlineAfter {
			offline = append(offline, *d)
		}
	}
	m := g.mqtt
	g.mu.RUnlock()
	for _, d := range offline {
		m.Offline(d)
	}
}

func (g *Gateway) Store() *store.Store { return g.store }
func (g *Gateway) LearnedCells() int   { return g.geo.Count() }
