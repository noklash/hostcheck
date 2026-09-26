# Network Routing Experiments

These experiments investigate Linux routing behavior before implementing route collection in Hostcheck.

The commands are diagnostic experiments only. Hostcheck does not parse their output in production.

## 1. Inspect the Routing Table

Commands:

```bash
ip route
ip route show default
```

Observed IPv4 state:

```text
default via 10.0.2.2 dev enp0s3 proto dhcp src 10.0.2.15 metric 100
10.0.2.0/24 dev enp0s3 proto kernel scope link src 10.0.2.15 metric 100
```

The host has:

* a default route through `10.0.2.2`
* a directly connected `10.0.2.0/24` route
* `enp0s3` as the output interface
* `10.0.2.15` as the preferred source
* metric `100`

## 2. Inspect Kernel Route Files

Commands:

```bash
ls -l /proc/net/route /proc/net/ipv6_route

cat /proc/net/route
cat /proc/net/ipv6_route
```

`/proc/net/route` exposes IPv4 routing information in a textual representation containing hexadecimal address fields.

Observed IPv4 routes included:

```text
enp0s3  00000000  0202000A  0003 ... 100  00000000
enp0s3  0002000A  00000000  0001 ... 100  00FFFFFF
```

These correspond to the default route and the directly connected `10.0.2.0/24` route.

`/proc/net/ipv6_route` exposes IPv6 routes using a lower-level textual representation.

The files were useful for understanding the kernel's route representation but are not selected as the production interface for Hostcheck.

## 3. Inspect IPv4 Route Selection

Commands:

```bash
ip route get 10.0.2.20
ip route get 1.1.1.1
ip route get 127.0.0.1
```

### Local subnet

Observed:

```text
10.0.2.20 dev enp0s3 src 10.0.2.15
```

The destination matches:

```text
10.0.2.0/24
```

so the kernel sends traffic directly through `enp0s3`.

No gateway is required.

### Public IPv4

Observed:

```text
1.1.1.1 via 10.0.2.2 dev enp0s3 src 10.0.2.15
```

The destination does not match the directly connected network, so the kernel selects the default route.

### Loopback

Observed:

```text
local 127.0.0.1 dev lo src 127.0.0.1
```

The kernel identifies the destination as local and selects the loopback interface.

## 4. Inspect Network Interface Indexes

Command:

```bash
ip -j link
```

Observed:

```text
ifindex 1 -> lo
ifindex 2 -> enp0s3
```

Linux networking uses interface indexes internally.

A route can therefore identify its output interface using an interface index.

Hostcheck will resolve the index to the corresponding interface name when creating its own model.

## 5. Inspect Structured IPv4 Routes

Command:

```bash
ip -j route
```

Observed:

```json
[
  {
    "dst": "default",
    "gateway": "10.0.2.2",
    "dev": "enp0s3",
    "protocol": "dhcp",
    "prefsrc": "10.0.2.15",
    "metric": 100,
    "flags": []
  },
  {
    "dst": "10.0.2.0/24",
    "dev": "enp0s3",
    "protocol": "kernel",
    "scope": "link",
    "prefsrc": "10.0.2.15",
    "metric": 100,
    "flags": []
  }
]
```

This demonstrates that a route can contain:

* destination
* gateway
* output interface
* protocol
* preferred source
* metric
* scope
* flags

The JSON output is useful for validation.

Hostcheck does not parse this JSON in production.

## 6. Inspect Structured IPv6 Routes

Command:

```bash
ip -j -6 route
```

Observed:

```json
[
  {
    "dst": "fd17:625c:f037:2::/64",
    "dev": "enp0s3",
    "protocol": "ra",
    "metric": 100,
    "flags": [],
    "pref": "medium"
  },
  {
    "dst": "fe80::/64",
    "dev": "enp0s3",
    "protocol": "kernel",
    "metric": 1024,
    "flags": [],
    "pref": "medium"
  },
  {
    "dst": "default",
    "gateway": "fe80::2",
    "dev": "enp0s3",
    "protocol": "ra",
    "metric": 20100,
    "flags": [],
    "pref": "medium"
  }
]
```

Important observation:

The IPv6 default gateway is:

```text
fe80::2
```

which is a link-local address.

IPv6 therefore cannot be modeled using IPv4-only gateway assumptions.

## 7. Inspect Detailed Routes

Commands:

```bash
ip -details route
ip -details -6 route
```

Observed IPv4:

```text
unicast default via 10.0.2.2 dev enp0s3 proto dhcp scope global src 10.0.2.15 metric 100
unicast 10.0.2.0/24 dev enp0s3 proto kernel scope link src 10.0.2.15 metric 100
```

Observed IPv6:

```text
unicast fd17:625c:f037:2::/64 dev enp0s3 proto ra scope global metric 100 pref medium
unicast fe80::/64 dev enp0s3 proto kernel scope global metric 1024 pref medium
unicast default via fe80::2 dev enp0s3 proto ra scope global metric 20100 pref medium
```

This confirms that route type, scope, protocol, metric, source, gateway, and output interface are distinct attributes.

## 8. Inspect Routing Rules

Commands:

```bash
ip rule
ip -6 rule
```

Observed IPv4:

```text
0:      from all lookup local
32766:  from all lookup main
32767:  from all lookup default
```

Observed IPv6:

```text
0:      from all lookup local
32766:  from all lookup main
```

Routing rules determine which routing tables are consulted.

The current host uses a conventional configuration.

Hostcheck V1 does not implement a complete policy-routing collector.

## 9. IPv6 Route Selection

Commands:

```bash
ip -j -6 route get ::1
ip -j -6 route get fd17:625c:f037:2::10
ip -j -6 route get 2001:4860:4860::8888
```

### IPv6 loopback

Observed:

```json
{
  "type": "local",
  "dst": "::1",
  "from": "::",
  "dev": "lo",
  "table": "local",
  "protocol": "kernel",
  "prefsrc": "::1",
  "metric": 0
}
```

The kernel identifies `::1` as a local route.

### IPv6 ULA

Observed:

```json
{
  "dst": "fd17:625c:f037:2::10",
  "from": "::",
  "dev": "enp0s3",
  "protocol": "ra",
  "prefsrc": "fd17:625c:f037:2:a00:27ff:fed0:dc4d",
  "metric": 100
}
```

The destination matches the directly routed IPv6 prefix.

No gateway is required.

### Public IPv6

Observed:

```json
{
  "dst": "2001:4860:4860::8888",
  "from": "::",
  "gateway": "fe80::2",
  "dev": "enp0s3",
  "protocol": "ra",
  "prefsrc": "fd17:625c:f037:2:a00:27ff:fed0:dc4d",
  "metric": 20100
}
```

The kernel selects the IPv6 default route through `fe80::2`.

## 10. Compare Routing With Address Configuration

Earlier address experiments showed:

```text
enp0s3:
    10.0.2.15/24
    fd17:625c:f037:2:a00:27ff:fed0:dc4d/64
    fe80::a00:27ff:fed0:dc4d/64
```

The routing experiments showed:

```text
IPv4:
    10.0.2.0/24 -> enp0s3
    default -> 10.0.2.2 -> enp0s3

IPv6:
    fd17:625c:f037:2::/64 -> enp0s3
    fe80::/64 -> enp0s3
    default -> fe80::2 -> enp0s3
```

This demonstrates that addresses and routes are related but separate kernel objects.

An address being configured does not mean Hostcheck should reconstruct routing state from the address.

## 11. Important Findings

### Routing is destination-dependent

Different destinations can select different routes.

The same interface can therefore carry traffic using different routing decisions.

### Direct routes do not require gateways

A destination matching a directly connected prefix can be sent through the interface without a next-hop gateway.

### IPv6 gateways can be link-local

The host's IPv6 default gateway is:

```text
fe80::2
```

Hostcheck must preserve gateway addresses without imposing IPv4 assumptions.

### Interface state does not prove connectivity

The host has:

```text
enp0s3 = UP
carrier = 1
address = configured
route = present
```

That does not by itself prove that a remote destination is reachable.

### Route configuration does not prove route success

A configured default route demonstrates kernel configuration.

It does not prove:

* gateway reachability
* DNS resolution
* Internet reachability
* application connectivity
* acceptable latency
* absence of packet loss

### Linux owns route selection

The kernel performs route selection.

Hostcheck should report kernel state instead of implementing another routing algorithm.

### `ip` output is useful for validation, not production parsing

The `ip` command provides convenient human-readable and JSON representations.

Hostcheck should not depend on parsing the command's output.

### `/proc` route files are useful for learning

`/proc/net/route` and `/proc/net/ipv6_route` expose useful kernel state, but their textual representations are not the preferred application interface for Hostcheck.

### Interface indexes matter

Linux identifies network interfaces internally using numeric indexes.

The collector must correctly map route output-interface indexes to interface names.

## 12. Implementation Direction

The production route collector will use Linux netlink/rtnetlink through a Go netlink library.

The Hostcheck model will remain independent of the library's internal route representation.

Conceptually:

```text
Linux kernel
     |
     v
rtnetlink
     |
     v
Go netlink library
     |
     v
Hostcheck Route model
     |
     v
network collector
```

The collector will query existing route state only.

It will not modify system routing configuration.

## 13. V1 Non-Goals

The routing subsystem does not currently attempt to:

* parse `ip` command output
* parse `/proc/net/route` in production
* parse `/proc/net/ipv6_route` in production
* reimplement kernel route selection
* modify routes
* modify routing rules
* perform gateway probes
* perform DNS checks
* measure latency
* measure packet loss
* run traceroute
* inspect firewall configuration
* inspect NAT configuration
* analyze network namespaces
* analyze container networking
* provide complete policy-routing analysis

## 14. Next Implementation Step

The next implementation step is to inspect and use the Go netlink API for route enumeration.

The collector should obtain the host's configured routes, translate Linux/netlink attributes into Hostcheck's own route model, and preserve meaningful missing values rather than fabricating them.

Validation will compare Hostcheck's observations against the previously recorded `ip -j route` and `ip -j -6 route` results.

EOF
