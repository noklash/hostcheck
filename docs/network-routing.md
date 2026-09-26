# Network Routing

## Purpose

Hostcheck needs to understand the host's routing state without reimplementing Linux route selection or parsing human-readable command output.

This document records the Linux routing model observed during development and defines the routing information Hostcheck V1 collects.

## Linux Routing Model

A Linux host can have multiple routes for different destination prefixes and address families.

A route can describe:

* destination prefix
* next-hop gateway
* output interface
* preferred source address
* metric
* protocol
* scope
* routing table
* route type

Routing rules determine which routing table is consulted. The selected route is then used by the kernel to determine how traffic should leave the host.

Hostcheck observes this state. It does not implement its own routing algorithm.

## Lab Host IPv4 State

The lab host currently has:

```text
default via 10.0.2.2 dev enp0s3 proto dhcp src 10.0.2.15 metric 100
10.0.2.0/24 dev enp0s3 proto kernel scope link src 10.0.2.15 metric 100
```

The default route sends destinations that do not match a more specific route through `10.0.2.2`.

The `10.0.2.0/24` route is directly connected through `enp0s3`, so destinations in that network do not require a gateway.

## Lab Host IPv6 State

The lab host currently has:

```text
fd17:625c:f037:2::/64 dev enp0s3 proto ra metric 100
fe80::/64 dev enp0s3 proto kernel metric 1024
default via fe80::2 dev enp0s3 proto ra metric 20100
```

The IPv6 default gateway is the link-local address `fe80::2`.

This demonstrates that a gateway should not be modeled with IPv4-specific assumptions.

## Route Selection Experiments

### Local IPv4 Destination

```bash
ip route get 10.0.2.20
```

Observed:

```text
10.0.2.20 dev enp0s3 src 10.0.2.15
```

The destination belongs to the directly connected `10.0.2.0/24` network.

No gateway is required.

### Public IPv4 Destination

```bash
ip route get 1.1.1.1
```

Observed:

```text
1.1.1.1 via 10.0.2.2 dev enp0s3 src 10.0.2.15
```

The destination does not match the directly connected network, so the kernel selects the IPv4 default route.

### IPv4 Loopback Destination

```bash
ip route get 127.0.0.1
```

Observed:

```text
local 127.0.0.1 dev lo src 127.0.0.1
```

The destination is local to the host and uses the loopback interface.

### IPv6 Loopback Destination

```bash
ip -j -6 route get ::1
```

Observed:

```text
type=local
dst=::1
dev=lo
table=local
protocol=kernel
prefsrc=::1
```

The kernel identifies `::1` as a local destination.

### IPv6 Directly Routed Destination

```bash
ip -j -6 route get fd17:625c:f037:2::10
```

Observed:

```text
dst=fd17:625c:f037:2::10
dev=enp0s3
prefsrc=fd17:625c:f037:2:a00:27ff:fed0:dc4d
metric=100
protocol=ra
```

The destination matches the IPv6 prefix routed through `enp0s3`.

No gateway is required.

### IPv6 Public Destination

```bash
ip -j -6 route get 2001:4860:4860::8888
```

Observed:

```text
dst=2001:4860:4860::8888
gateway=fe80::2
dev=enp0s3
prefsrc=fd17:625c:f037:2:a00:27ff:fed0:dc4d
metric=20100
protocol=ra
```

The kernel selects the IPv6 default route through `fe80::2`.

## Routing Rules

The host currently uses the following IPv4 rules:

```text
0:      from all lookup local
32766:  from all lookup main
32767:  from all lookup default
```

IPv6 currently reports:

```text
0:      from all lookup local
32766:  from all lookup main
```

Routing rules are a separate layer from routes.

Conceptually:

```text
packet
  |
  v
routing rules
  |
  v
routing table
  |
  v
selected route
```

Hostcheck V1 does not attempt to model the complete policy-routing rule system.

## Interface Indexes

Linux networking identifies interfaces internally using interface indexes.

The lab host currently has:

```text
1 -> lo
2 -> enp0s3
```

A route may therefore identify an output interface by its numeric index rather than its human-readable name.

Hostcheck resolves the kernel interface index to the corresponding interface name when building its own model.

## Route Attributes

The routing information exposed by Linux includes several attributes.

### Address Family

Identifies whether the route belongs to IPv4 or IPv6.

### Destination

The destination prefix matched by the route.

Examples:

```text
0.0.0.0/0
10.0.2.0/24
::/0
fd17:625c:f037:2::/64
```

### Gateway

The next-hop address when the route requires one.

A directly connected route may have no gateway.

### Output Interface

The interface through which packets leave the host.

### Preferred Source

The source address Linux prefers when traffic uses the route.

### Metric

A route preference value used when routes are otherwise applicable.

Hostcheck records the metric rather than attempting to reproduce the kernel's route-selection algorithm.

### Protocol

Identifies how the route was installed or originated.

Observed examples include:

```text
kernel
dhcp
ra
```

### Scope

Describes the route's scope.

For example, the directly connected IPv4 route is reported with:

```text
scope link
```

Hostcheck should preserve Linux-provided scope information rather than infer it from Go IP classification functions.

### Route Type

Routes may have different types.

For example:

```text
unicast
local
```

The IPv6 loopback lookup demonstrated a `local` route.

### Routing Table

Routes belong to routing tables.

The normal host configuration uses the `local`, `main`, and potentially `default` tables through routing rules.

## Kernel Interfaces

### `ip route`

The `ip` command is useful for human inspection and validation.

Hostcheck does not parse its output.

### `/proc/net/route`

Linux exposes IPv4 routing information through:

```text
/proc/net/route
```

The representation contains hexadecimal destination and gateway values along with route metadata.

This is useful for understanding the kernel's representation but is not the production interface used by Hostcheck.

### `/proc/net/ipv6_route`

Linux exposes IPv6 routing information through:

```text
/proc/net/ipv6_route
```

Its representation is lower-level and less convenient for application-level collection.

Hostcheck does not parse this file for production route collection.

### rtnetlink

Linux provides structured networking information through netlink.

The routing-specific interface is rtnetlink.

Conceptually:

```text
Hostcheck
    |
    v
rtnetlink
    |
    v
Linux networking subsystem
```

This allows userspace programs to query structured kernel networking state without executing and parsing the `ip` command.

Hostcheck uses a Go netlink library as its implementation boundary.

## Why Hostcheck Does Not Parse `ip`

The `ip` command is a diagnostic and administrative interface intended primarily for users.

Parsing its text output introduces dependencies on:

* command availability
* output formatting
* field ordering
* version-specific presentation
* text parsing rules

Earlier address experiments demonstrated this problem.

A fixed field position in `ip` output was incorrectly assumed to contain address scope, but an IPv4 broadcast address appeared before the scope field.

Structured kernel networking interfaces avoid this class of parsing error.

## Hostcheck V1 Design

Hostcheck V1 collects configured routing state through Linux's native networking interface.

The internal Hostcheck model remains independent of the netlink library's internal route representation.

The model preserves information useful for understanding host routing:

* address family
* destination prefix
* gateway when present
* output interface
* preferred source when present
* metric
* protocol
* scope when available
* route type
* routing table when available

The collector preserves absence as absence.

For example:

* a directly connected route has no gateway
* a route may have no preferred source
* some attributes are not meaningful for every route

Hostcheck must not replace missing information with fabricated values.

## Routing and Connectivity Are Different

A configured route does not prove that traffic can successfully reach its destination.

These observations are independent:

```text
interface state
    |
    v
address configuration
    |
    v
routing configuration
    |
    v
actual connectivity
```

For example, a host can have:

```text
interface = UP
address   = configured
route     = present
gateway   = configured
```

while the gateway or remote destination is unreachable.

Hostcheck therefore treats route collection as observation of configuration and kernel state.

Active connectivity testing is a separate capability.

## V1 Scope

Hostcheck V1 collects routing state.

It does not:

* reimplement Linux route selection
* modify routes
* modify routing rules
* perform gateway probes
* perform DNS checks
* measure latency
* measure packet loss
* run traceroute
* inspect firewall configuration
* inspect NAT configuration
* analyze container networking
* analyze network namespaces
* provide a complete policy-routing analyzer

## Lab Environment Limitations

The current experiments were performed on a simple VirtualBox Linux host with:

* one network namespace
* one Ethernet interface
* one loopback interface
* IPv4 networking
* IPv6 networking

More complex systems may contain:

* multiple interfaces
* multiple default routes
* policy routing
* VRFs
* bridges
* bonds
* tunnels
* network namespaces
* containers
* multiple next hops

These are outside the current V1 scope.

## Important Engineering Findings

### Finding 1: Routing is destination-dependent

Different destinations can select different routes on the same host and interface.

### Finding 2: Direct routes do not require gateways

A destination matching a directly connected route can be reached through the output interface without a gateway.

### Finding 3: IPv6 gateways can be link-local

The lab host uses:

```text
fe80::2
```

as its IPv6 default gateway.

### Finding 4: Address classification is not Linux scope

Go's `net.IP` classification functions do not provide Linux route/address scope.

Linux-provided scope should therefore be used where scope semantics matter.

### Finding 5: Route configuration does not prove connectivity

A route is evidence of configured kernel state, not proof that packets can successfully reach a destination.

### Finding 6: Linux already owns route selection

Hostcheck should observe and report the kernel's routing state rather than implement another routing algorithm.

## Conclusion

The routing experiments established that Linux routing is a kernel-managed system with multiple address families, routing tables, routing rules, route types, metrics, protocols, scopes, gateways, and source-address information.

Hostcheck V1 therefore uses structured Linux networking information rather than parsing command output or legacy `/proc` route files.

The collector reports meaningful kernel state while leaving route selection and packet forwarding to Linux itself.