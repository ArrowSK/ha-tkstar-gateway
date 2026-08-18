package mqtt

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/ArrowSK/ha-tkstar-gateway/internal/model"
	paho "github.com/eclipse/paho.mqtt.golang"
)

type PrivacyHandler func(key string, enabled bool)

type Client struct {
	c         paho.Client
	log       *slog.Logger
	onPrivacy PrivacyHandler
}

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
}

func New(cfg Config, log *slog.Logger, onPrivacy PrivacyHandler) *Client {
	broker := fmt.Sprintf("tcp://%s:%d", cfg.Host, cfg.Port)
	opts := paho.NewClientOptions().
		AddBroker(broker).
		SetClientID("tkstar-gateway").
		SetAutoReconnect(true).
		SetConnectRetry(false).
		SetKeepAlive(30 * time.Second).
		SetPingTimeout(5 * time.Second)
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}

	m := &Client{log: log, onPrivacy: onPrivacy}
	opts.OnConnect = func(c paho.Client) {
		log.Info("MQTT connected")
		t := c.Subscribe("tkstar_gateway/+/privacy/set", 0, m.command)
		if !t.WaitTimeout(10*time.Second) || t.Error() != nil {
			log.Warn("MQTT privacy command subscription failed")
		}
	}
	opts.OnConnectionLost = func(_ paho.Client, err error) {
		log.Warn("MQTT disconnected", "error", err)
	}
	m.c = paho.NewClient(opts)
	return m
}

func (m *Client) Start() error {
	t := m.c.Connect()
	if !t.WaitTimeout(20 * time.Second) {
		return fmt.Errorf("MQTT connect timeout")
	}
	return t.Error()
}

func (m *Client) command(_ paho.Client, msg paho.Message) {
	p := strings.Split(msg.Topic(), "/")
	if len(p) != 4 {
		return
	}
	v := strings.ToUpper(strings.TrimSpace(string(msg.Payload())))
	if m.onPrivacy != nil {
		m.onPrivacy(p[1], v == "ON" || v == "TRUE" || v == "1")
	}
}

func (m *Client) pub(topic string, payload any, retain bool) {
	var b []byte
	switch v := payload.(type) {
	case string:
		b = []byte(v)
	case []byte:
		b = v
	default:
		b, _ = json.Marshal(v)
	}
	t := m.c.Publish(topic, 0, retain, b)
	if !t.WaitTimeout(5*time.Second) || t.Error() != nil {
		m.log.Debug("MQTT publish failed", "topic", topic)
	}
}

func (m *Client) Discover(d model.Device) {
	k := d.Key
	base := "tkstar_gateway/" + k
	dev := map[string]any{
		"identifiers":  []string{"tkstar_gateway_" + k},
		"name":         d.Name,
		"manufacturer": "TKSTAR/WINNES-compatible",
		"model":        d.Model,
	}

	m.pub("homeassistant/device_tracker/tkstar_"+k+"/config", map[string]any{
		"name":                  "Location",
		"unique_id":             "tkstar_" + k + "_location",
		"json_attributes_topic": base + "/position",
		"availability_topic":    base + "/location_availability",
		"source_type":           "gps",
		"device":                dev,
	}, true)

	sensor := func(n, label, unit, icon, deviceClass string) {
		v := map[string]any{
			"name":        label,
			"unique_id":   "tkstar_" + k + "_" + n,
			"state_topic": base + "/" + n,
			"device":      dev,
		}
		if unit != "" {
			v["unit_of_measurement"] = unit
		}
		if icon != "" {
			v["icon"] = icon
		}
		if deviceClass != "" {
			v["device_class"] = deviceClass
		}
		m.pub("homeassistant/sensor/tkstar_"+k+"_"+n+"/config", v, true)
	}

	sensor("battery", "Battery", "%", "", "battery")
	sensor("speed", "Speed", "km/h", "mdi:speedometer", "")
	sensor("source", "Position source", "", "mdi:crosshairs-gps", "")
	sensor("signal", "GSM signal", "%", "mdi:signal", "")
	sensor("satellites", "Satellites", "", "mdi:satellite-variant", "")
	sensor("heading", "Direction", "°", "mdi:compass", "")
	sensor("status", "Tracking status", "", "mdi:car-connected", "")
	sensor("last_update", "Last update", "", "", "timestamp")

	m.pub("homeassistant/binary_sensor/tkstar_"+k+"_online/config", map[string]any{
		"name":               "Online",
		"unique_id":          "tkstar_" + k + "_online",
		"state_topic":        base + "/availability",
		"payload_on":         "online",
		"payload_off":        "offline",
		"device_class":       "connectivity",
		"device":             dev,
	}, true)

	m.pub("homeassistant/switch/tkstar_"+k+"_privacy/config", map[string]any{
		"name":          "Privacy mode",
		"unique_id":     "tkstar_" + k + "_privacy",
		"state_topic":   base + "/privacy/state",
		"command_topic": base + "/privacy/set",
		"payload_on":     "ON",
		"payload_off":    "OFF",
		"device":         dev,
	}, true)

	if d.Privacy {
		m.pub(base+"/privacy/state", "ON", true)
	} else {
		m.pub(base+"/privacy/state", "OFF", true)
	}
}

func (m *Client) State(d model.Device) {
	base := "tkstar_gateway/" + d.Key
	m.pub(base+"/availability", "online", true)

	if d.Privacy {
		m.pub(base+"/location_availability", "offline", true)
		m.pub(base+"/privacy/state", "ON", true)
		m.pub(base+"/status", "Privacy", true)
		return
	}

	m.pub(base+"/privacy/state", "OFF", true)
	m.pub(base+"/battery", strconv.FormatFloat(d.Battery, 'f', 1, 64), true)
	m.pub(base+"/signal", strconv.Itoa(d.GSM), true)

	if d.Last == nil || !d.Last.Valid {
		m.pub(base+"/location_availability", "offline", true)
		m.pub(base+"/status", "No position", true)
		return
	}

	p := d.Last
	m.pub(base+"/location_availability", "online", true)
	m.pub(base+"/position", map[string]any{
		"latitude":        p.Latitude,
		"longitude":       p.Longitude,
		"gps_accuracy":    p.Accuracy,
		"position_source": p.Source,
		"speed_kmh":       p.SpeedKmh,
		"heading":         p.Heading,
		"battery_level":   p.Battery,
		"protocol":        p.Protocol,
	}, true)
	m.pub(base+"/speed", strconv.FormatFloat(p.SpeedKmh, 'f', 1, 64), true)
	m.pub(base+"/source", p.Source, true)
	m.pub(base+"/satellites", strconv.Itoa(p.Satellites), true)
	m.pub(base+"/heading", strconv.FormatFloat(p.Heading, 'f', 0, 64), true)
	if p.SpeedKmh > 1 {
		m.pub(base+"/status", "Moving", true)
	} else {
		m.pub(base+"/status", "Stopped", true)
	}
	m.pub(base+"/last_update", p.ReceivedAt.Format(time.RFC3339), true)
}

func (m *Client) Offline(d model.Device) {
	base := "tkstar_gateway/" + d.Key
	m.pub(base+"/availability", "offline", true)
	m.pub(base+"/location_availability", "offline", true)
	if !d.Privacy {
		m.pub(base+"/status", "Offline", true)
	}
}
