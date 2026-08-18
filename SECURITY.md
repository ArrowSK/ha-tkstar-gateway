# Security

TKSTAR Gateway accepts raw tracker traffic from the Internet when you forward its tracker ports through your router. Treat those ports as untrusted network inputs.

The App rejects unknown tracker IDs by default. New trackers are accepted only while pairing mode is explicitly enabled from the authenticated Home Assistant Ingress UI. Pairing automatically closes after ten minutes.

Do not publish tracker IDs, SIM numbers, ICCIDs, exact coordinates, packet captures, API keys, or router configuration in public issues.

The Ingress web interface is intended to be reached through Home Assistant. Do not separately expose the Ingress port.
