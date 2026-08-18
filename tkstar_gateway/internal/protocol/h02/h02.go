package h02

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/ArrowSK/ha-tkstar-gateway/internal/model"
	"github.com/ArrowSK/ha-tkstar-gateway/internal/protocol"
)

type Server struct{ Addr string }

func (s Server) Name() string { return "h02" }

func (s Server) Run(ctx context.Context, sink protocol.Sink, log *slog.Logger) error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	log.Info("tracker listener started", "protocol", s.Name(), "addr", s.Addr)
	for {
		c, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go handle(c, sink, log)
	}
}

func handle(c net.Conn, sink protocol.Sink, log *slog.Logger) {
	defer c.Close()
	r := bufio.NewReaderSize(c, 4096)
	for {
		_ = c.SetReadDeadline(time.Now().Add(3 * time.Minute))
		raw, err := r.ReadString('#')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Debug("h02 connection ended", "error", err)
			}
			return
		}
		m, err := Parse(strings.TrimSpace(raw))
		if err != nil {
			log.Debug("unsupported h02 packet", "error", err)
			continue
		}
		if !sink.Accept(m.DeviceID, "h02") {
			return
		}
		sink.Handle(m)
	}
}

func Parse(raw string) (model.Message, error) {
	if len(raw) < 2 || raw[0] != '*' || raw[len(raw)-1] != '#' {
		return model.Message{}, errors.New("bad framing")
	}
	d := strings.Split(raw[1:len(raw)-1], ",")
	if len(d) < 12 || d[0] != "HQ" || d[2] != "V1" {
		return model.Message{}, errors.New("unsupported packet")
	}

	p := &model.Position{
		DeviceID:   d[1],
		Protocol:   "h02",
		ReceivedAt: time.Now().UTC(),
		Valid:      strings.EqualFold(d[4], "A"),
		Source:     "GPS",
		Battery:    -1,
		GSM:        -1,
	}
	p.Timestamp, _ = time.ParseInLocation("020106150405", d[11]+d[3], time.UTC)

	var err error
	if p.Latitude, err = ddm(d[5], 2, strings.EqualFold(d[6], "S")); err != nil {
		return model.Message{}, err
	}
	if p.Longitude, err = ddm(d[7], 3, strings.EqualFold(d[8], "W")); err != nil {
		return model.Message{}, err
	}
	if v, e := strconv.ParseFloat(d[9], 64); e == nil {
		p.SpeedKmh = v * 1.852
	}
	p.Heading, _ = strconv.ParseFloat(d[10], 64)

	var b *float64
	if len(d) > 17 {
		if n, e := strconv.Atoi(d[17]); e == nil {
			v := battery(n)
			p.Battery = v
			b = &v
		}
	}
	if !p.Valid {
		p.Source = "UNKNOWN"
		p.Latitude = 0
		p.Longitude = 0
	}
	return model.Message{DeviceID: d[1], Protocol: "h02", Position: p, Battery: b}, nil
}

func ddm(s string, n int, neg bool) (float64, error) {
	if len(s) <= n {
		return 0, errors.New("invalid coordinate")
	}
	d, err := strconv.ParseFloat(s[:n], 64)
	if err != nil {
		return 0, err
	}
	m, err := strconv.ParseFloat(s[n:], 64)
	if err != nil {
		return 0, err
	}
	v := d + m/60
	if neg {
		v = -v
	}
	return v, nil
}

func battery(n int) float64 {
	switch {
	case n <= 0:
		return 0
	case n <= 3:
		return float64((n - 1) * 10)
	case n <= 6:
		return float64((n - 1) * 20)
	case n <= 100:
		return float64(n)
	case n >= 0xF1 && n <= 0xF6:
		v := float64(n-0xF0) * 20
		if v > 100 {
			return 100
		}
		return v
	default:
		return 0
	}
}
