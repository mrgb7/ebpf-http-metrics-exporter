package main

import (
	"encoding/binary"
	"net/netip"
	"strconv"

	"github.com/cilium/ebpf"
	"github.com/prometheus/client_golang/prometheus"
)

var responsesDesc = prometheus.NewDesc(
	"http_responses_total",
	"HTTP responses seen by eBPF, by client, server and status code.",
	[]string{"client_namespace", "client_workload", "server_namespace", "server_workload", "code"},
	nil,
)

type collector struct {
	responses *ebpf.Map
	pods      *podCache
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) { ch <- responsesDesc }

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	totals := map[[5]string]uint64{}

	var key bpfResponseKey
	var count uint64
	entries := c.responses.Iterate()
	for entries.Next(&key, &count) {
		client := c.pods.lookup(toIP(key.ClientIp))
		server := c.pods.lookup(toIP(key.ServerIp))
		labels := [5]string{
			client.namespace, client.workload,
			server.namespace, server.workload,
			strconv.Itoa(int(key.Status)),
		}
		totals[labels] += count
	}

	for labels, total := range totals {
		ch <- prometheus.MustNewConstMetric(responsesDesc, prometheus.CounterValue, float64(total), labels[:]...)
	}
}

func toIP(raw uint32) string {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], raw)
	return netip.AddrFrom4(b).String()
}
