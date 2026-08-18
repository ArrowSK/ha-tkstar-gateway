package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ArrowSK/ha-tkstar-gateway/internal/gateway"
	"github.com/ArrowSK/ha-tkstar-gateway/internal/geolocation"
	"github.com/ArrowSK/ha-tkstar-gateway/internal/mqtt"
	"github.com/ArrowSK/ha-tkstar-gateway/internal/protocol"
	"github.com/ArrowSK/ha-tkstar-gateway/internal/protocol/gt06"
	"github.com/ArrowSK/ha-tkstar-gateway/internal/protocol/h02"
	"github.com/ArrowSK/ha-tkstar-gateway/internal/protocol/watch"
	"github.com/ArrowSK/ha-tkstar-gateway/internal/store"
	"github.com/ArrowSK/ha-tkstar-gateway/internal/webui"
)

const version = "0.1.0"

func main() {
	self := flag.Bool("self-test", false, "run startup self-test and exit")
	flag.Parse()
	if *self {
		fmt.Println("TKSTAR Gateway self-test OK", version)
		return
	}

	level := slog.LevelInfo
	switch strings.ToLower(env("TKSTAR_LOG_LEVEL", "info")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	data := env("TKSTAR_DATA_DIR", "/data")
	st, err := store.New(data, envInt("TKSTAR_HISTORY_DAYS", 180))
	fatal(log, "store", err)
	_ = st.Cleanup(time.Now())

	geo := geolocation.New(data, os.Getenv("TKSTAR_GOOGLE_GEOLOCATION_API_KEY"))
	gw := gateway.New(
		st,
		geo,
		log,
		time.Duration(envInt("TKSTAR_OFFLINE_AFTER_MINUTES", 10))*time.Minute,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	mc := mqtt.New(mqtt.Config{
		Host:     env("TKSTAR_MQTT_HOST", "core-mosquitto"),
		Port:     envInt("TKSTAR_MQTT_PORT", 1883),
		Username: os.Getenv("TKSTAR_MQTT_USERNAME"),
		Password: os.Getenv("TKSTAR_MQTT_PASSWORD"),
	}, log, gw.SetPrivacyByKey)

	go func() {
		for ctx.Err() == nil {
			if err := mc.Start(); err == nil {
				gw.SetMQTT(mc)
				return
			} else {
				log.Warn("MQTT not ready; tracker receiver remains available", "error", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	}()

	servers := []protocol.Server{
		watch.Server{Addr: env("TKSTAR_WATCH_ADDR", ":5093")},
		h02.Server{Addr: env("TKSTAR_H02_ADDR", ":5013")},
		gt06.Server{Addr: env("TKSTAR_GT06_ADDR", ":5023")},
	}
	for _, srv := range servers {
		s := srv
		go func() {
			if err := s.Run(ctx, gw, log); err != nil && ctx.Err() == nil {
				log.Error("tracker listener stopped", "protocol", s.Name(), "error", err)
				cancel()
			}
		}()
	}

	go func() {
		if err := webui.Listen(
			env("TKSTAR_HTTP_ADDR", ":8099"),
			webui.New(gw, log).Handler(),
			log,
		); err != nil && ctx.Err() == nil {
			log.Error("web server stopped", "error", err)
			cancel()
		}
	}()

	offlineTick := time.NewTicker(time.Minute)
	cleanupTick := time.NewTicker(6 * time.Hour)
	defer offlineTick.Stop()
	defer cleanupTick.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("stopping")
			return
		case <-offlineTick.C:
			gw.MarkOffline()
		case <-cleanupTick.C:
			_ = st.Cleanup(time.Now())
		}
	}
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func envInt(k string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(k))
	if err == nil && v > 0 {
		return v
	}
	return fallback
}

func fatal(log *slog.Logger, what string, err error) {
	if err != nil {
		log.Error(what+" failed", "error", err)
		os.Exit(1)
	}
}
