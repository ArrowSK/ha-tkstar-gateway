package protocol

import (
 "context"
 "log/slog"
 "github.com/ArrowSK/ha-tkstar-gateway/internal/model"
)

type Sink interface { Accept(deviceID, protocol string) bool; Handle(model.Message) }
type Server interface { Name() string; Run(context.Context, Sink, *slog.Logger) error }
