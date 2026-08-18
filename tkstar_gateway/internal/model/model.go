package model

import "time"

type Cell struct { MCC int `json:"mcc"`; MNC int `json:"mnc"`; LAC int `json:"lac"`; ID int `json:"id"`; Signal int `json:"signal,omitempty"` }

type Position struct {
 DeviceID string `json:"device_id,omitempty"`
 Timestamp time.Time `json:"timestamp"`
 ReceivedAt time.Time `json:"received_at"`
 Latitude float64 `json:"latitude"`
 Longitude float64 `json:"longitude"`
 Valid bool `json:"valid"`
 Source string `json:"source"`
 Accuracy float64 `json:"accuracy,omitempty"`
 SpeedKmh float64 `json:"speed_kmh,omitempty"`
 Heading float64 `json:"heading,omitempty"`
 Altitude float64 `json:"altitude,omitempty"`
 Satellites int `json:"satellites,omitempty"`
 GSM int `json:"gsm,omitempty"`
 Battery float64 `json:"battery,omitempty"`
 Protocol string `json:"protocol"`
 Alarm string `json:"alarm,omitempty"`
 Cells []Cell `json:"cells,omitempty"`
}

type Message struct { DeviceID string; Protocol string; Position *Position; Battery *float64; GSM *int; ICCID string; Alarm string }

type Device struct {
 ID string `json:"id"`
 Key string `json:"key"`
 Name string `json:"name"`
 Model string `json:"model,omitempty"`
 SIM string `json:"sim,omitempty"`
 ICCID string `json:"iccid,omitempty"`
 Plate string `json:"plate,omitempty"`
 Icon string `json:"icon,omitempty"`
 Protocol string `json:"protocol,omitempty"`
 Privacy bool `json:"privacy"`
 CreatedAt time.Time `json:"created_at"`
 LastSeen time.Time `json:"last_seen,omitempty"`
 Battery float64 `json:"battery,omitempty"`
 GSM int `json:"gsm,omitempty"`
 Last *Position `json:"last_position,omitempty"`
}
