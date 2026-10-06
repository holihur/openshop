// Package metrics provides a Prometheus implementation of port.Metrics.
//
// Metric vectors are created lazily from the first observation so application
// code can pass arbitrary label maps without pre-declaring every metric. Label
// names are captured on first use; later calls fill missing labels with "".
package metrics

import (
	"net/http"
	"sort"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/holihur/openshop/internal/port"
)

// Prometheus implements port.Metrics.
type Prometheus struct {
	registry *prometheus.Registry

	mu        sync.Mutex
	counters  map[string]*counterMetric
	gauges    map[string]*gaugeMetric
	histogram map[string]*histogramMetric
}

type counterMetric struct {
	names []string
	vec   *prometheus.CounterVec
}

type gaugeMetric struct {
	names []string
	vec   *prometheus.GaugeVec
}

type histogramMetric struct {
	names []string
	vec   *prometheus.HistogramVec
}

func New() *Prometheus {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return &Prometheus{
		registry:  registry,
		counters:  map[string]*counterMetric{},
		gauges:    map[string]*gaugeMetric{},
		histogram: map[string]*histogramMetric{},
	}
}

func (p *Prometheus) Counter(name string, delta float64, labels map[string]string) {
	p.mu.Lock()
	m, ok := p.counters[name]
	if !ok {
		names := labelNames(labels)
		m = &counterMetric{
			names: names,
			vec:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: name, Help: name}, names),
		}
		p.registry.MustRegister(m.vec)
		p.counters[name] = m
	}
	p.mu.Unlock()
	m.vec.With(selectLabels(m.names, labels)).Add(delta)
}

func (p *Prometheus) Gauge(name string, value float64, labels map[string]string) {
	p.mu.Lock()
	m, ok := p.gauges[name]
	if !ok {
		names := labelNames(labels)
		m = &gaugeMetric{
			names: names,
			vec:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name, Help: name}, names),
		}
		p.registry.MustRegister(m.vec)
		p.gauges[name] = m
	}
	p.mu.Unlock()
	m.vec.With(selectLabels(m.names, labels)).Set(value)
}

func (p *Prometheus) Histogram(name string, value float64, labels map[string]string) {
	p.mu.Lock()
	m, ok := p.histogram[name]
	if !ok {
		names := labelNames(labels)
		m = &histogramMetric{
			names: names,
			vec: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{Name: name, Help: name, Buckets: prometheus.DefBuckets},
				names,
			),
		}
		p.registry.MustRegister(m.vec)
		p.histogram[name] = m
	}
	p.mu.Unlock()
	m.vec.With(selectLabels(m.names, labels)).Observe(value)
}

// Handler serves the Prometheus exposition format.
func (p *Prometheus) Handler() http.Handler {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})
}

func labelNames(labels map[string]string) []string {
	names := make([]string, 0, len(labels))
	for k := range labels {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func selectLabels(names []string, labels map[string]string) prometheus.Labels {
	if len(names) == 0 {
		return prometheus.Labels{}
	}
	out := make(prometheus.Labels, len(names))
	for _, n := range names {
		out[n] = labels[n]
	}
	return out
}

var _ port.Metrics = (*Prometheus)(nil)
