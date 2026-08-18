# Architecture

The App separates transport/protocol handling from tracker state, position resolution, persistence, MQTT and presentation.

- `internal/protocol` emits normalized messages.
- `internal/gateway` owns pairing, tracker metadata, privacy enforcement and ingest.
- `internal/geolocation` learns cell locations from valid GPS fixes and may use an optional external fallback.
- `internal/store` persists metadata and daily JSONL history without a database daemon.
- `internal/mqtt` publishes Home Assistant MQTT Discovery and current state.
- `internal/webui` serves the Ingress interface and JSON API.

Raw tracker IDs are used only inside the App data directory. MQTT topics and Home Assistant unique IDs use a SHA-256-derived local key.
