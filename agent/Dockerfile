FROM golang:1.27 AS build
RUN apt-get update && apt-get install -y --no-install-recommends clang llvm libbpf-dev bpftool
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# vmlinux.h describes the kernel's types; CO-RE adapts them at load time.
RUN bpftool btf dump file /sys/kernel/btf/vmlinux format c > bpf/vmlinux.h \
 && go generate \
 && CGO_ENABLED=0 go build -o /agent .

FROM gcr.io/distroless/static
COPY --from=build /agent /agent
ENTRYPOINT ["/agent"]
