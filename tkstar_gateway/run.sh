#!/usr/bin/with-contenv bashio
set -euo pipefail

export TKSTAR_DATA_DIR=/data
export TKSTAR_HTTP_ADDR=:8099
export TKSTAR_WATCH_ADDR=:5093
export TKSTAR_H02_ADDR=:5013
export TKSTAR_GT06_ADDR=:5023
export TKSTAR_HISTORY_DAYS="$(bashio::config 'history_days')"
export TKSTAR_OFFLINE_AFTER_MINUTES="$(bashio::config 'offline_after_minutes')"
export TKSTAR_GOOGLE_GEOLOCATION_API_KEY="$(bashio::config 'google_geolocation_api_key')"
export TKSTAR_LOG_LEVEL="$(bashio::config 'log_level')"
export TKSTAR_MQTT_HOST="$(bashio::services mqtt 'host')"
export TKSTAR_MQTT_PORT="$(bashio::services mqtt 'port')"
export TKSTAR_MQTT_USERNAME="$(bashio::services mqtt 'username')"
export TKSTAR_MQTT_PASSWORD="$(bashio::services mqtt 'password')"
exec /usr/local/bin/tkstar-gateway
