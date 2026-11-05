package main

// Compiles bpf/http.c and generates the Go bindings (bpfObjects,
// loadBpfObjects, bpfResponseKey). The kprobe uses BPF_KPROBE, which needs
// an architecture-specific target. vmlinux.h is dumped from the build host's
// kernel, so only the build architecture ($GOARCH) can be compiled.
//go:generate go tool bpf2go -tags linux -target $GOARCH -type response_key bpf bpf/http.c
