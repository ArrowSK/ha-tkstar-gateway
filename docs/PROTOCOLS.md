# Protocol support

## Watch — port 5093

Text frames use `[vendor*device*hex_length*content]`. The Gateway handles heartbeat (`LK`), ICCID (`CCID`), GPS/location (`UD`, `UD2`) and alarm (`AL`) messages. `LK` and `AL` are acknowledged.

Watch location reports can include GPS, battery, GSM and neighbouring cell information. Invalid GPS reports are passed to the LBS resolver.

## H02 — port 5013

The initial implementation handles common ASCII `*HQ,<id>,V1,...#` location reports and extracts DDM coordinates, speed, heading and battery.

## GT06 — port 5023

The initial implementation handles login, location, status and alarm frames using `0x7878` framing and CRC-ITU. Login/status/alarm frames are acknowledged where required.

## Adding protocols

Implement the `protocol.Server` contract and normalize data into `model.Message`. Keep vendor-specific parsing inside the protocol package.
