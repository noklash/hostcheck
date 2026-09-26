# Network Route Table Experiment

This experiment inspects `/proc/net/route` and decodes the raw IPv4 route representation exposed by the Linux kernel.

It exists for first-principles learning and validation. It is **not** the production route collector for Hostcheck.

## Objective

Understand how Linux represents IPv4 routes at the `/proc` interface and compare that low-level representation with the semantic route information exposed by `iproute2`.

The experiment focuses on:

* route destination
* gateway
* network mask
* interface
* flags
* metric
* byte-order representation

No route selection logic is implemented.

## Observed Kernel Representation

On the current VM:

```text
Iface   Destination     Gateway         Flags   RefCnt  Use     Metric  Mask
enp0s3  00000000        0202000A        0003    0       0       100     00000000
enp0s3  0002000A        00000000        0001    0       0       100     00FFFFFF
```

The IPv4 address fields are represented in little-endian byte order on this x86 system.

The raw entries therefore decode to:

```text
default route:
  destination = 0.0.0.0
  gateway     = 10.0.2.2
  interface   = enp0s3
  mask        = 0.0.0.0
  metric      = 100

connected route:
  destination = 10.0.2.0
  gateway     = 0.0.0.0
  interface   = enp0s3
  mask        = 255.255.255.0
  metric      = 100
```

The zero gateway on the connected route represents the fact that traffic to `10.0.2.0/24` does not require a gateway.

## Comparison With iproute2

The same IPv4 routing table is presented by `iproute2` as:

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

The two views describe the same routing configuration, but at different abstraction levels.

`/proc/net/route` exposes a lower-level kernel representation.

`iproute2` presents that information in a more semantic form.

For example, `iproute2` identifies:

* the default route
* the gateway
* the output interface
* the route protocol
* the preferred source address
* the route scope
* the metric

Those concepts are not all represented directly in the same form in `/proc/net/route`.

## What the Experiment Demonstrates

### 1. A route is more than a destination and gateway

A useful route model may contain several independent attributes:

```text
destination
gateway
output interface
preferred source
metric
protocol
scope
route type
routing table
```

These fields should not be collapsed into a single concept.

### 2. A gateway is not required for every route

The connected route:

```text
10.0.2.0/24 dev enp0s3
```

does not have a gateway.

The default route:

```text
default via 10.0.2.2 dev enp0s3
```

does.

This distinction matters when modeling routes.

### 3. Raw kernel representations may require decoding

The `/proc/net/route` address values are not displayed as normal dotted IPv4 addresses.

For example:

```text
0202000A
```

represents:

```text
10.0.2.2
```

on the current little-endian system.

Therefore, a parser must understand the representation rather than treating the raw field as a normal IP address string.

### 4. Configuration does not prove connectivity

The presence of:

```text
default via 10.0.2.2 dev enp0s3
```

proves that Linux has a configured default route.

It does **not** prove that:

* the gateway is reachable
* DNS works
* the Internet is reachable
* an application can establish a connection
* packets are successfully leaving the host

Hostcheck must keep configuration state and observed connectivity as separate concepts.

## IPv6

Linux also exposes IPv6 routing information through:

```text
/proc/net/ipv6_route
```

The current system reports entries such as:

```text
fd17:625c:f037:2::/64
fe80::/64
default via fe80::2
```

The raw representation is considerably less readable than the IPv4 `/proc/net/route` format and contains several kernel-specific fields.

The current experiment intentionally does not implement an IPv6 `/proc` parser.

The IPv6 output is retained as an observation for later work.

## Why This Is Not the Production Route Collector

Hostcheck should not parse `ip` command output for production data collection.

`iproute2` is an excellent diagnostic and validation tool, but its human-facing output is not the API that Hostcheck should depend on.

`/proc/net/route` is useful for understanding Linux's low-level representation, but it is also not the intended long-term production interface for Hostcheck route collection.

The production implementation should use Linux's native routing interface, **rtnetlink/netlink**, when route collection is implemented.

That approach should allow Hostcheck to consume structured kernel route information instead of reverse-engineering command output.

## Scope of This Experiment

This experiment intentionally does **not** implement:

* route selection
* policy routing
* multiple routing tables
* VRFs
* network namespaces
* IPv6 route parsing
* connectivity probing
* gateway health checks
* DNS checks
* latency or packet-loss measurement
* production route collection

Linux remains responsible for route selection.

Hostcheck should observe and report relevant routing state rather than reimplementing the kernel's routing logic.

## Result

This experiment established the distinction between four different layers:

```text
Linux kernel routing state
        ↓
/proc/net/route
        ↓
iproute2 interpretation
        ↓
Hostcheck's future structured model
```

The main engineering conclusion is:

> `/proc/net/route` is valuable for understanding how Linux exposes routing internally, but Hostcheck should eventually consume structured rtnetlink information rather than parse either `/proc` route text or `ip` command output.

The next production networking step is therefore to study rtnetlink and determine the minimum route metadata Hostcheck actually needs for V1.