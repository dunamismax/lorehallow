package session

import (
	"strings"

	"github.com/dunamismax/lorehallow/server/internal/metrics"
	"github.com/dunamismax/lorehallow/server/internal/pb"
)

// Metrics are the connection metrics, shared by every session.
type Metrics struct {
	Open         *metrics.Gauge
	Closed       *metrics.Counter
	Evicted      *metrics.Counter
	MessagesIn   *metrics.Counter
	MessagesOut  *metrics.Counter
	BytesIn      *metrics.Counter
	BytesOut     *metrics.Counter
	RateDropped  *metrics.Counter
	InputDropped *metrics.Counter
	RTT          *metrics.Histogram

	reasons map[pb.DisconnectReason]*metrics.Counter
}

// NewMetrics registers the connection metrics in r (a private registry
// when r is nil).
func NewMetrics(r *metrics.Registry) *Metrics {
	if r == nil {
		r = metrics.NewRegistry()
	}
	m := &Metrics{
		Open:         r.NewGauge("lorehallow_connections_open", "Open game connections, in any state."),
		Closed:       r.NewCounter("lorehallow_connections_closed_total", "Game connections closed, for any reason."),
		Evicted:      r.NewCounter("lorehallow_connections_evicted_total", "Connections closed because their send queue filled (too slow)."),
		MessagesIn:   r.NewCounter("lorehallow_messages_in_total", "WebSocket messages received from clients."),
		MessagesOut:  r.NewCounter("lorehallow_messages_out_total", "WebSocket messages written to clients."),
		BytesIn:      r.NewCounter("lorehallow_bytes_in_total", "Payload bytes received from clients."),
		BytesOut:     r.NewCounter("lorehallow_bytes_out_total", "Payload bytes written to clients (before WebSocket framing)."),
		RateDropped:  r.NewCounter("lorehallow_messages_rate_dropped_total", "Client messages dropped by the inbound rate limit."),
		InputDropped: r.NewCounter("lorehallow_inputs_queue_dropped_total", "Inputs dropped because the player's input queue was full."),
		RTT:          r.NewHistogram("lorehallow_heartbeat_rtt_seconds", "Round-trip time of server Pings, measured from client Pongs.", metrics.DurationBuckets()),
		reasons:      map[pb.DisconnectReason]*metrics.Counter{},
	}
	for v, name := range pb.DisconnectReason_name {
		reason := strings.ToLower(strings.TrimPrefix(name, "DISCONNECT_REASON_"))
		m.reasons[pb.DisconnectReason(v)] = r.NewCounter("lorehallow_disconnects_"+reason+"_total", "Connections closed with reason "+reason+" (unspecified: the client closed or the socket failed).")
	}
	return m
}
