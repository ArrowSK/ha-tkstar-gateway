package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ArrowSK/ha-tkstar-gateway/internal/model"
)

type Store struct {
	root      string
	retention int
	mu        sync.Mutex
}

func New(root string, retention int) (*Store, error) {
	if err := os.MkdirAll(root, 0750); err != nil {
		return nil, err
	}
	return &Store{root: root, retention: retention}, nil
}

func (s *Store) LoadDevices() ([]model.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(filepath.Join(s.root, "devices.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var devices []model.Device
	err = json.Unmarshal(b, &devices)
	return devices, err
}

func (s *Store) SaveDevices(devices []model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sort.Slice(devices, func(i, j int) bool { return devices[i].Name < devices[j].Name })
	b, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.root, "devices.json.tmp")
	if err = os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.root, "devices.json"))
}

func safe(value string) string {
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}

func (s *Store) AppendPosition(key string, position model.Position) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.append("history", key, position)
}

func (s *Store) AppendAlarm(key string, position model.Position) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.append("alarms", key, position)
}

func (s *Store) append(kind, key string, position model.Position) error {
	day := position.ReceivedAt.UTC().Format("2006-01-02")
	dir := filepath.Join(s.root, kind, safe(key))
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}
	b, err := json.Marshal(position)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, day+".jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func (s *Store) Positions(key string, from, to time.Time, max int) ([]model.Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if max <= 0 {
		max = 5000
	}
	var out []model.Position
	for day := dateOnly(from); !day.After(dateOnly(to)); day = day.AddDate(0, 0, 1) {
		err := scan(filepath.Join(s.root, "history", safe(key), day.Format("2006-01-02")+".jsonl"), func(p model.Position) {
			if !p.ReceivedAt.Before(from) && !p.ReceivedAt.After(to) {
				out = append(out, p)
			}
		})
		if err != nil {
			return nil, err
		}
	}
	if len(out) <= max || max == 1 {
		if max == 1 && len(out) > 0 {
			return out[len(out)-1:], nil
		}
		return out, nil
	}
	step := float64(len(out)-1) / float64(max-1)
	sample := make([]model.Position, 0, max)
	for i := 0; i < max; i++ {
		sample = append(sample, out[int(math.Round(float64(i)*step))])
	}
	return sample, nil
}

func (s *Store) Alarms(key string, from, to time.Time, max int) ([]model.Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.Position
	for day := dateOnly(from); !day.After(dateOnly(to)); day = day.AddDate(0, 0, 1) {
		err := scan(filepath.Join(s.root, "alarms", safe(key), day.Format("2006-01-02")+".jsonl"), func(p model.Position) {
			if p.Alarm != "" && !p.ReceivedAt.Before(from) && !p.ReceivedAt.After(to) {
				out = append(out, p)
			}
		})
		if err != nil {
			return nil, err
		}
	}
	if max > 0 && len(out) > max {
		out = out[len(out)-max:]
	}
	return out, nil
}

func scan(path string, fn func(model.Position)) error {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var p model.Position
		if json.Unmarshal(scanner.Bytes(), &p) == nil {
			fn(p)
		}
	}
	return scanner.Err()
}

func dateOnly(value time.Time) time.Time {
	u := value.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

func (s *Store) Cleanup(now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := dateOnly(now.UTC().AddDate(0, 0, -s.retention))
	for _, kind := range []string{"history", "alarms"} {
		base := filepath.Join(s.root, kind)
		deviceDirs, _ := os.ReadDir(base)
		for _, deviceDir := range deviceDirs {
			if !deviceDir.IsDir() {
				continue
			}
			files, _ := os.ReadDir(filepath.Join(base, deviceDir.Name()))
			for _, file := range files {
				if file.IsDir() || !strings.HasSuffix(file.Name(), ".jsonl") {
					continue
				}
				day, err := time.Parse("2006-01-02", strings.TrimSuffix(file.Name(), ".jsonl"))
				if err == nil && day.Before(cutoff) {
					_ = os.Remove(filepath.Join(base, deviceDir.Name(), file.Name()))
				}
			}
		}
	}
	return nil
}

type Stats struct {
	Points      int       `json:"points"`
	DistanceKm  float64   `json:"distance_km"`
	MaxSpeedKmh float64   `json:"max_speed_kmh"`
	First       time.Time `json:"first,omitempty"`
	Last        time.Time `json:"last,omitempty"`
}

func ComputeStats(positions []model.Position) Stats {
	var stats Stats
	var previous *model.Position
	for i := range positions {
		p := positions[i]
		if !p.Valid {
			continue
		}
		stats.Points++
		if stats.First.IsZero() {
			stats.First = p.ReceivedAt
		}
		stats.Last = p.ReceivedAt
		if p.SpeedKmh > stats.MaxSpeedKmh {
			stats.MaxSpeedKmh = p.SpeedKmh
		}
		if previous != nil && p.ReceivedAt.Sub(previous.ReceivedAt) <= 30*time.Minute {
			distance := haversine(previous.Latitude, previous.Longitude, p.Latitude, p.Longitude)
			if distance < 100 {
				stats.DistanceKm += distance
			}
		}
		copy := p
		previous = &copy
	}
	return stats
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
