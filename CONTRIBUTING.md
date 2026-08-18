# Contributing

Contributions are welcome, especially protocol samples sanitised of tracker IDs, SIM/ICCID values and exact private coordinates.

Protocol additions should be isolated under `internal/protocol`, include tests, and normalise data into the shared model rather than coupling protocol-specific fields to the web or MQTT layers.

Run before submitting:

```sh
cd tkstar_gateway
go test ./...
go vet ./...
go build ./cmd/tkstar-gateway
```

Never commit packet captures or real tracker credentials.
