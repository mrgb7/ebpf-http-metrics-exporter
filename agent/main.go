package main

import (
	"log"
	"net/http"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
	}

	var objs bpfObjects
	if err := loadBpfObjects(&objs, nil); err != nil {
		log.Fatalf("loading eBPF: %v", err)
	}
	defer objs.Close()

	probe, err := link.Kprobe("tcp_sendmsg", objs.CountResponse, nil)
	if err != nil {
		log.Fatalf("attaching kprobe: %v", err)
	}
	defer probe.Close()

	pods := newPodCache()
	prometheus.MustRegister(&collector{responses: objs.Responses, pods: pods})

	log.Println("counting HTTP responses, metrics on :9100/metrics")
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9100", nil))
}
