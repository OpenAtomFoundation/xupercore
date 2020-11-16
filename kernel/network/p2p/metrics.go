package p2p

import prom "github.com/prometheus/client_golang/prometheus"

var (
	// Metrics is the default instance of metrics. It is intended
	// to be used in conjunction the default Prometheus metrics
	// registry.
	Metrics = newMetrics()
)

var (
	qps = prom.NewCounterVec(
		prom.CounterOpts{
			Name: "qps",
			Help: "p2p server qps",
		},
		[]string{"bcname", "type", "method"})
	cost = prom.NewCounterVec(
		prom.CounterOpts{
			Name: "cost",
			Help: "p2p server cost",
		},
		[]string{"bcname", "type", "method"})
	packet = prom.NewCounterVec(
		prom.CounterOpts{
			Name: "packet",
			Help: "p2p server packet",
		},
		[]string{"bcname", "type", "method"})
)

// metrics is the metrics of p2p server
type metrics struct {
	QPS    *prom.CounterVec
	Cost   *prom.CounterVec
	Packet *prom.CounterVec
}

func newMetrics() *metrics {
	return &metrics{
		QPS:    qps,
		Cost:   cost,
		Packet: packet,
	}
}

func init() {
	prom.MustRegister(qps)
	prom.MustRegister(cost)
	prom.MustRegister(packet)
}
