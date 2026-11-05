# ebpf-http-metrics-exporter

> **Work in progress.** This project is under active development — the metric
> set, labels and deployment layout may change without notice. Not ready for
> production use.

A Prometheus metrics exporter for HTTP traffic between Kubernetes pods, built
on eBPF. It counts responses by status code using only a kernel probe, with no
changes to the apps. A Go agent turns the pod IPs into Kubernetes names.

```
shop ──HTTP──▶ orders      orders answers 70% 200 · 15% 404 · 10% 500 · 5% 503
                 │
     kernel: tcp_sendmsg() ── eBPF reads "HTTP/1.1 404" ── map[server, client, code]++
                 │
  Go agent: pod IP → namespace/app (watch on API server) ── :9100/metrics
```

Today the exporter emits one metric, `http_responses_total`. The direction is a
broader HTTP metrics exporter (latency, request rates) — see **Limits**.

## Run

```sh
make up        # k3d cluster + build images + deploy
make metrics   # wait ~20s first
make down
```

```
http_responses_total{client_workload="shop",server_workload="orders",code="200"} 80
http_responses_total{client_workload="shop",server_workload="orders",code="404"} 15
http_responses_total{client_workload="shop",server_workload="orders",code="500"} 8
```

## Layout

| File | Role |
|---|---|
| `agent/bpf/http.c` | kprobe on `tcp_sendmsg`, counts response status lines |
| `agent/main.go` | loads the probe, serves metrics |
| `agent/pods.go` | pod IP → workload cache (client-go informer) |
| `agent/metrics.go` | reads the eBPF map on each scrape |
| `app/main.go` | demo server and traffic client |

## Limits (on purpose)

- Plain HTTP/1.x over IPv4 only. No TLS, no HTTP/2/gRPC.
- Only looks at the first bytes of each write. A status line split across writes is missed.
- Workload name is the pod's `app` label. Unknown IPs (node, host network) show as `unknown`.
- One k3d node only: k3d nodes share one kernel, so a second agent would double-count.

## Browse metrics

```sh
make ui        # Prometheus on http://localhost:9090
```

Useful queries:

```promql
# requests per second by status code
sum by (code) (rate(http_responses_total[1m]))

# error ratio (5xx) per client → server
sum by (client_workload, server_workload) (rate(http_responses_total{code=~"5.."}[1m]))
  / sum by (client_workload, server_workload) (rate(http_responses_total[1m]))
```
