<h1 align="center">TKSTAR Gateway</h1>

<p align="center"><strong>A lightweight Home Assistant-native tracking server for TKSTAR/WINNES-style GPS trackers.</strong></p>

<p align="center"><a href="https://github.com/ArrowSK/ha-tkstar-gateway/actions/workflows/ci.yaml"><img src="https://github.com/ArrowSK/ha-tkstar-gateway/actions/workflows/ci.yaml/badge.svg?branch=main" alt="CI"></a> <a href="https://github.com/ArrowSK/ha-tkstar-gateway/actions/workflows/builder.yaml"><img src="https://github.com/ArrowSK/ha-tkstar-gateway/actions/workflows/builder.yaml/badge.svg?branch=main" alt="Builder"></a> <img src="https://img.shields.io/badge/Home%20Assistant-App-41BDF5?logo=home-assistant&logoColor=white" alt="Home Assistant app"> <img src="https://img.shields.io/badge/License-MIT-blue" alt="MIT"></p>

<p align="center"><a href="https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2FArrowSK%2Fha-tkstar-gateway"><img src="https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg" alt="Add repository" height="40"></a></p>

> **Experimental 0.1.0.** Do not change a working tracker's server address until the App is installed, running, and pairing mode has been opened.

## What it is

TKSTAR Gateway receives tracker packets directly instead of polling an undocumented vendor web API or running a full fleet-management stack. It is designed for small Home Assistant installations that want live tracking, route history, alarms and automations without a Java/database fleet platform.

The App is multi-tracker from the start. Tracker identity is taken from the wire protocol, not from one configured device.

## 0.1.0 feature baseline

| Capability | Status |
|---|---|
| Multiple trackers | Yes |
| Watch protocol | Yes |
| H02 protocol | Yes |
| GT06 protocol | Yes |
| GPS positions | Yes |
| Watch cell/LBS observations | Yes |
| Learned local cell positioning | Yes |
| Optional Google Geolocation fallback | Yes |
| Live map | Yes |
| Route playback/history | Yes |
| Distance/max-speed statistics | Yes |
| Alarm log | Yes |
| Device details/name/model/SIM/ICCID/plate | Yes |
| Home Assistant MQTT Discovery | Yes |
| Per-tracker privacy mode | Yes |
| Unknown tracker rejection + timed pairing | Yes |

Support is protocol-based. A model name beginning with `TK` does not guarantee a particular wire protocol, so additional protocol modules can be added without changing the rest of the App.

## Architecture

```text
Tracker(s)
   │ cellular TCP
   ▼
TKSTAR Gateway
   ├── Watch / H02 / GT06 decoders
   ├── tracker registry + timed pairing
   ├── GPS / local LBS / optional Google LBS resolver
   ├── lightweight daily JSONL history
   ├── Ingress live map + playback + stats + alarms
   └── MQTT Discovery
             │
             ▼
        Home Assistant
```

There is no separate database server. History is partitioned by tracker and day under the App's persistent `/data` directory.

## Installation

1. Add this repository to **Settings → Apps → App store → Repositories**.
2. Install **TKSTAR Gateway**.
3. Start it and open the sidebar page.
4. Press **Pair new tracker (10 min)**.
5. Only then point the tracker at your public hostname/IP and the correct TCP port.
6. Forward that TCP port on the router to the Home Assistant host.

Default listener ports are Watch `5093/tcp`, H02 `5013/tcp`, and GT06 `5023/tcp`. They are configurable in the App Network section.

## Pairing and privacy

Unknown tracker IDs are rejected by default. The authenticated Ingress UI can open a ten-minute pairing window. A tracker that connects during that window is registered locally; later unknown IDs remain rejected.

Privacy mode is per tracker. While enabled, protocol acknowledgements continue so the physical tracker does not enter a retry storm, but new location telemetry is discarded: it is not stored, externally resolved, or published to Home Assistant. Existing history is not automatically deleted.

## LBS positioning

Watch packets can carry MCC, MNC, LAC and cell IDs even when GPS is invalid. The Gateway first learns tower locations when a valid GPS fix and cell observation arrive together. Later LBS-only reports can use that local cache without an external lookup.

An optional Google Geolocation API key can be configured in App options. External LBS lookup is disabled when the key is empty, and is never called while that tracker is in privacy mode.

## Resource design

The runtime is one Go process. It uses no Java VM, PostgreSQL, PostGIS, Redis or separate web server. The browser renders the map; the Home Assistant host stores only normalized tracker events and history. History retention defaults to 180 days and cleanup is automatic.

## Attribution

Protocol handling was bootstrapped from the MIT-licensed [`freman/gps2mqtt`](https://github.com/freman/gps2mqtt) project. The original copyright and licence are preserved in [THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md).

This project is independent and is not affiliated with TKSTAR or WINNES.
