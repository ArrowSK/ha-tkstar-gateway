# Privacy model

Tracker IDs and exact positions stay inside the installed App and local MQTT broker. Home Assistant MQTT identifiers and topics use a SHA-256-derived local key rather than the raw tracker ID.

When privacy mode is enabled for a tracker:

- tracker protocol acknowledgements continue;
- new coordinates are discarded;
- no route-history row is written;
- no cell tower is learned;
- no external geolocation request is made;
- the Home Assistant location entity is marked unavailable;
- non-location connectivity can still be shown in the App.

Existing route history is not automatically deleted.
