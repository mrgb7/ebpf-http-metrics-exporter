
#include "vmlinux.h"
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

#define AF_INET 2
#define AF_INET6 10

struct response_key {
  __u32 server_ip;
  __u32 client_ip;
  __u16 status;
  __u16 _pad;
};

struct {
  __uint(type, BPF_MAP_TYPE_HASH);
  __uint(max_entries, 10240);
  __type(key, struct response_key);
  __type(value, __u64);
} responses SEC(".maps");

static void *payload_start(struct msghdr *msg) {
  __u8 type = BPF_CORE_READ(msg, msg_iter.iter_type);

  if (type == bpf_core_enum_value(enum iter_type, ITER_UBUF))
    return BPF_CORE_READ(msg, msg_iter.ubuf);
  if (type == bpf_core_enum_value(enum iter_type, ITER_IOVEC))
    return BPF_CORE_READ(msg, msg_iter.__iov, iov_base);
  return NULL;
}

static bool is_digit(char c) { return c >= '0' && c <= '9'; }

static bool ipv4_addrs(struct sock *sk, __u32 *local, __u32 *remote) {
  __u16 family = BPF_CORE_READ(sk, __sk_common.skc_family);

  if (family == AF_INET) {
    *local = BPF_CORE_READ(sk, __sk_common.skc_rcv_saddr);
    *remote = BPF_CORE_READ(sk, __sk_common.skc_daddr);
    return true;
  }
  if (family == AF_INET6) {
    __u32 l[4], r[4];
    BPF_CORE_READ_INTO(&l, sk, __sk_common.skc_v6_rcv_saddr.in6_u.u6_addr32);
    BPF_CORE_READ_INTO(&r, sk, __sk_common.skc_v6_daddr.in6_u.u6_addr32);
    bool mapped = l[0] == 0 && l[1] == 0 && l[2] == bpf_htonl(0xffff);
    *local = l[3];
    *remote = r[3];
    return mapped;
  }
  return false;
}

SEC("kprobe/tcp_sendmsg")
int BPF_KPROBE(count_response, struct sock *sk, struct msghdr *msg) {
  char line[12]; // "HTTP/1.1 200"
  void *buf = payload_start(msg);
  if (!buf || bpf_probe_read_user(line, sizeof(line), buf) < 0)
    return 0;

  if (__builtin_memcmp(line, "HTTP/1.", 7) != 0 || line[8] != ' ')
    return 0;
  if (!is_digit(line[9]) || !is_digit(line[10]) || !is_digit(line[11]))
    return 0;

  struct response_key k = {
      .status =
          (line[9] - '0') * 100 + (line[10] - '0') * 10 + (line[11] - '0'),
  };
  if (!ipv4_addrs(sk, &k.server_ip, &k.client_ip))
    return 0;

  __u64 one = 1;
  __u64 *count = bpf_map_lookup_elem(&responses, &k);
  if (count)
    __sync_fetch_and_add(count, 1);
  else
    bpf_map_update_elem(&responses, &k, &one, BPF_NOEXIST);
  return 0;
}

char LICENSE[] SEC("license") = "GPL";
