# Changelog

## 0.1.0

Initial experimental release.

- Home Assistant App packaging with Ingress UI and multi-architecture images.
- Multi-tracker registry with secure ten-minute pairing mode and unknown-device rejection.
- Direct TCP receiver for Watch, H02 and GT06 protocol families.
- Watch GPS, battery, GSM, alarm and cell/LBS parsing.
- Local learned cell-tower positioning and optional Google Geolocation fallback.
- Optional per-tracker LBS filtering.
- Lightweight partitioned JSONL history, route playback, statistics and alarms.
- Per-tracker overspeed threshold with five-minute alarm debounce.
- Monitor UI with All/Active/Inactive filters, moving/stopped/offline status, speed, direction, battery and position source.
- MQTT Discovery for Home Assistant location, tracking status, battery, speed, direction, signal, position source and privacy entities.
- Per-tracker privacy mode that continues protocol acknowledgements but discards new location telemetry and keeps connectivity separate from location availability.
