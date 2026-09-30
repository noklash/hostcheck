# Network Routing

## Purpose

Hostcheck needs to observe the host's routing state without reimplementing Linux route selection or parsing human-readable command output.

This document records the Linux routing model observed during development and defines the routing information Hostcheck V1 currently collects.

Hostcheck observes routing configuration exposed by the Linux kernel. It does not implement an independent routing algorithm and does not treat the presence of a route as proof of network connectivity.

## Linux Routing Model

A Linux host can have multiple routes for different destination prefixes and address families.

A route can describe:

* destination prefix
* source prefix
* next-hop gateway
* output interface
* route priority
* protocol
* scope
* routing table
* route type
* route flags

Linux routing also involves routing rules, multiple routing tables, policy routing, and other mechanisms.

Hostcheck observes the route state exposed by the kernel. It does not attempt to reproduce the complete kernel route-selection process.

Conceptually:

```text
packet
   |
   v
routing rules
   |
   v
routing table(s)
   |
   v
candidate routes
   |
   v
Linux route selection
   |
   v
selected route
```

The exact selection process remains the responsibility of the Linux kernel.

## Lab Host IPv4 State

The lab host currently has:

```text
default via 10.0.2.2 dev enp0s3 proto dhcp src 10.0.2.15 metric 100

10.0.2.0/24 dev enp0s3 proto kernel scope link src 10.0.2.15 metric 100
```

The default route handles destinations that do not match a more specific applicable route.

The `10.0.2.0/24` route represents the directly connected IPv4 network.

The default route uses:

```text
gateway = 10.0.2.2
interface = enp0s3
priority/metric = 100
protocol = DHCP
```

The connected route has no gateway because the destination network is directly reachable through `enp0s3`.

## Lab Host IPv6 State

The lab host currently has:

```text
fd17:625c:f037:2::/64 dev enp0s3 proto ra metric 100

fe80::/64 dev enp0s3 proto kernel metric 1024

default via fe80::2 dev enp0s3 proto ra metric 20100
```

The IPv6 default gateway is the link-local address:

```text
fe80::2
```

This demonstrates that a gateway cannot be modeled using IPv4-specific assumptions.

A gateway is simply an IP address associated with a route and may belong to either address family.

## Route Selection Experiments

The experiments used the `ip route get` interface to observe decisions made by the Linux kernel.

These commands are useful for understanding route selection, but Hostcheck does not execute them as part of route collection.

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

This also demonstrates that the complete kernel route state contains routes that are not necessarily shown by the simplest `ip route` output.

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

These experiments demonstrate an important distinction:

```text
route configuration

        !=

kernel route lookup result

        !=

actual network connectivity
```

Hostcheck currently focuses on observing the first of these.

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

A simplified representation is:

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
candidate route
  |
  v
selected route
```

Hostcheck V1 does not currently collect or model the complete policy-routing rule system.

The rules were inspected during experimentation to understand the relationship between routing tables and route entries, but they remain outside the current route collector.

## Interface Indexes

Linux networking identifies interfaces internally using numeric interface indexes.

The lab host currently has:

```text
1 -> lo

2 -> enp0s3
```

A rtnetlink route message therefore identifies an output interface using its numeric index.

The current Hostcheck route model preserves that kernel value:

```go
type Route struct {
    Family          uint8
    Destination     net.IP
    PrefixLen       uint8
    Source          net.IP
    SourcePrefixLen uint8
    Gateway         net.IP
    InterfaceIndex  uint32
    Priority        uint32
    Table           uint32
    Protocol        uint8
    Scope           uint8
    Type            uint8
    Flags           uint32
    Multipath       []NextHop
}
```

Hostcheck keeps `InterfaceIndex` as the raw kernel identifier in the low-level route model.

The route collector does not replace the kernel index with an interface name.

The higher-level `Network` snapshot collects interface identity separately. This allows consumers of the assembled network observation to correlate a route's `InterfaceIndex` with an `Interface.Index` and therefore obtain the corresponding interface name without changing the underlying route representation.

This separation keeps the low-level collector kernel-facing while allowing higher-level integration to provide human-readable context.

## Route Attributes

The rtnetlink route message exposes several pieces of information.

### Address Family

Identifies whether the route belongs to IPv4 or IPv6.

The current collector preserves the kernel family value.

Observed families are:

```text
IPv4

IPv6
```

### Destination

The destination prefix matched by the route.

Examples include:

```text
0.0.0.0/0

10.0.2.0/24

::/0

fd17:625c:f037:2::/64
```

The current model stores the destination IP and prefix length separately.

### Default Routes

A default route is represented by a zero destination prefix.

In the current rtnetlink representation, the destination may be absent while the prefix length is zero:

```text
Destination = nil

PrefixLen   = 0
```

This represents:

```text
0.0.0.0/0
```

for IPv4 or:

```text
::/0
```

for IPv6.

Hostcheck preserves this representation rather than inventing an explicit destination address.

### Source

Rtnetlink can expose a source prefix associated with a route.

The current model preserves:

```go
Source          net.IP
SourcePrefixLen uint8
```

The current route collector does not attempt to interpret this as the preferred source-address selection algorithm.

Source information is preserved as kernel route state.

### Gateway

The gateway is the next-hop address when the route requires one.

For example:

```text
IPv4:

    10.0.2.2

IPv6:

    fe80::2
```

A directly connected route may have no gateway.

The current collector preserves a missing gateway as `nil`.

It does not invent a gateway for routes that do not have one.

### Output Interface

The route identifies the interface through which traffic should leave the host.

Rtnetlink provides the output interface as a numeric kernel interface index.

The current model stores:

```go
InterfaceIndex uint32
```

For the lab host:

```text
2 -> enp0s3
```

The low-level route collector does not convert that index into the interface name.

The higher-level network snapshot provides the interface collection needed to correlate the index with interface identity.

### Priority

The route can contain a priority value, corresponding to the metric exposed by common Linux networking tools.

The lab host contains examples such as:

```text
IPv4 default:

    priority 100

IPv4 connected:

    priority 100

IPv6 RA network:

    priority 100

IPv6 link-local:

    priority 1024

IPv6 default:

    priority 20100
```

Hostcheck records the value.

It does not attempt to reproduce the kernel's route-selection algorithm.

### Protocol

The route protocol identifies how the route was installed or originated.

Observed values include:

```text
kernel

dhcp

ra
```

The rtnetlink API exposes these as numeric protocol values.

Hostcheck currently preserves the numeric kernel representation rather than converting it immediately into presentation strings.

For the lab host:

```text
16 -> DHCP

2  -> kernel

9  -> RA
```

The human-readable names are useful for documentation and inspection, but the numeric value is the underlying kernel state.

### Scope

Routes can contain a Linux scope value.

For example, the directly connected IPv4 route is observed with:

```text
scope = 253
```

which corresponds to link scope.

The current model preserves the numeric scope:

```go
Scope uint8
```

Hostcheck should preserve Linux-provided route scope rather than infer it from IP classification functions.

Route scope and address scope are related Linux networking concepts, but they are separate pieces of kernel state and should not be conflated.

### Route Type

Routes can have different types.

Observed examples include:

```text
unicast

local

broadcast
```

The current model preserves the numeric route type:

```go
Type uint8
```

The rtnetlink experiment demonstrated that the kernel exposes local and broadcast routes that are not necessarily visible in ordinary `ip route` output.

For example, the lab host exposes local-table routes such as:

```text
10.0.2.15/32

127.0.0.0/8

127.0.0.1/32
```

and broadcast routes such as:

```text
10.0.2.255/32

127.255.255.255/32
```

This is an important difference between observing the kernel's route state and parsing a simplified command view.

### Routing Table

Routes belong to routing tables.

The lab host exposes routes in at least:

```text
254 -> main

255 -> local
```

The rtnetlink experiment showed both ordinary main-table routes and local-table routes.

For example:

```text
table=254
```

was observed for the normal IPv4 and IPv6 routes.

Local and broadcast routes were observed with:

```text
table=255
```

The current `Route` model preserves the routing table as:

```go
Table uint32
```

The model therefore retains the kernel table identifier instead of assuming that every route belongs to the main table.

## Kernel Interfaces

### `ip route`

The `ip` command is useful for human inspection and validation.

For example:

```bash
ip route

ip -6 route

ip -j route

ip -j -6 route

ip route get <destination>
```

These commands were heavily used during development to understand Linux routing behavior.

Hostcheck does not parse their output.

### `/proc/net/route`

Linux exposes IPv4 routing information through:

```text
/proc/net/route
```

The representation contains hexadecimal destination and gateway values together with route metadata.

A dedicated experiment was used to inspect the raw IPv4 route table.

This was useful for understanding how Linux exposes routing state through `/proc`.

It is not the production route collection interface used by Hostcheck.

The experiment is retained as learning and validation evidence rather than becoming part of the production collector.

### `/proc/net/ipv6_route`

Linux also exposes IPv6 routing information through:

```text
/proc/net/ipv6_route
```

Its representation is lower-level and less convenient for application-level structured collection.

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
    |
    v
kernel route state
```

This allows userspace programs to query structured networking state without executing and parsing the `ip` command.

Hostcheck uses the Go `github.com/jsimonetti/rtnetlink` library as the implementation boundary.

The route collector requests the kernel route list and maps the returned rtnetlink messages into the Hostcheck `Route` model.

## Rtnetlink Route Experiment

The rtnetlink experiment was used to inspect the complete route messages exposed by the kernel.

The lab host produced entries including:

```text
family=IPv4

dst=<none>/0

table=254

protocol=16

scope=0

type=1

gateway=10.0.2.2

oif=2

priority=100
```

and:

```text
family=IPv4

dst=10.0.2.0/24

table=254

protocol=2

scope=253

type=1

gateway=<none>

oif=2

priority=100
```

IPv6 included:

```text
family=IPv6

dst=fd17:625c:f037:2::/64

table=254

protocol=9

scope=0

type=1

gateway=<none>

oif=2

priority=100
```

and:

```text
family=IPv6

dst=<none>/0

table=254

protocol=9

scope=0

type=1

gateway=fe80::2

oif=2

priority=20100
```

The experiment also exposed local and broadcast routes in table 255.

This established that rtnetlink provides a more complete representation of the kernel's route state than the simplified output normally shown by:

```bash
ip route
```

## Why Hostcheck Does Not Parse `ip`

The `ip` command is a diagnostic and administrative interface intended primarily for users.

Parsing its text output introduces dependencies on:

* command availability
* output formatting
* field ordering
* version-specific presentation
* text parsing rules

The address experiments already demonstrated how positional parsing can fail.

For example:

```text
... brd 10.0.2.255 scope global ...
```

contains the broadcast address before the scope value.

Structured kernel interfaces avoid this class of parsing problem.

The same principle applies to routes.

Hostcheck should consume structured kernel networking information directly rather than depend on the presentation format of the `ip` command.

## Hostcheck V1 Design

Hostcheck V1 collects configured routing state through Linux's native networking interface.

The current route model is:

```go
type Route struct {
    Family          uint8
    Destination     net.IP
    PrefixLen       uint8
    Source          net.IP
    SourcePrefixLen uint8
    Gateway         net.IP
    InterfaceIndex  uint32
    Priority        uint32
    Table           uint32
    Protocol        uint8
    Scope           uint8
    Type            uint8
    Flags           uint32
    Multipath       []NextHop
}
```

The model is intentionally close to the information exposed by rtnetlink.

It preserves:

* address family
* destination
* destination prefix length
* source
* source prefix length
* gateway
* output interface index
* route priority
* routing table
* protocol
* scope
* route type
* route flags
* multipath next hops

The collector preserves absence as absence.

For example:

* a directly connected route can have no gateway
* a route can have no source attribute
* a default route can have no destination address in the rtnetlink representation
* some attributes are not meaningful for every route

Hostcheck must not replace missing information with fabricated values.

## Route Flags

Rtnetlink also exposes route flags.

The current model preserves:

```go
Flags uint32
```

rather than translating the value into a reduced set of application-level booleans.

The current lab environment reports zero flags for the observed routes, but the field remains part of the model so that kernel-provided route state is not discarded.

Interpretation of individual flags can be added later if Hostcheck develops a requirement for them.

## Multipath Routes

Rtnetlink can represent multipath routes with multiple next hops.

The underlying library exposes multipath information through route attributes.

Hostcheck V1 preserves this information structurally through the `Route.Multipath` field:

```go
Multipath []NextHop
```

Each next hop preserves its kernel-provided interface index, gateway, hop value, and flags.

A multipath route is therefore not flattened into a single gateway.

The `Hops` and `Flags` values are preserved as raw kernel-provided values. Hostcheck does not reinterpret `Hops` as a routing weight or attempt to reproduce Linux's multipath selection behavior.

This keeps the collector faithful to the kernel representation while leaving higher-level interpretation for a later layer if it becomes necessary.

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

## Route Configuration and Route Selection Are Different

Hostcheck must also distinguish between observing routes and performing a route lookup.

The route collector reports the configured kernel route entries.

It does not ask:

```text
"Which route would Linux choose for 1.1.1.1?"
```

and attempt to reproduce that answer itself.

Linux already owns route selection.

The development experiments used:

```bash
ip route get
```

to observe the kernel's actual decisions.

This is useful for understanding the system, but it should not be replaced with a duplicate algorithm inside Hostcheck.

## Network Observation Snapshot

Hostcheck combines interface state, interface addresses, and routing information into a `Network` observation through `ReadNetwork()`.

The collection sequence is:

```text
enumerate interfaces
        |
        v
read interface state
        |
        v
read interface addresses
        |
        v
read routing state
        |
        v
assemble Network
```

The resulting `Network` model records:

```go
type Network struct {
    ObservedAt time.Time
    Interfaces []Interface
    Routes     []Route
}
```

`ObservedAt` records when the network observation cycle begins.

The resulting object is a best-effort observation, not an atomic kernel snapshot. The underlying information comes from separate Linux kernel interfaces and can change while collection is in progress.

For example, interface traffic counters can change between individual reads even when interface configuration, addresses, and routes remain unchanged.

Hostcheck therefore does not use locking, retries, or transaction-like mechanisms to create the appearance of atomicity across independent kernel interfaces.

The timestamp provides temporal context for the observation without claiming that every field was read at exactly the same instant.

Route interface indexes remain kernel-facing in the low-level route model. The higher-level network snapshot contains the corresponding interface indexes and names, allowing consumers to correlate a route's output interface with the collected interface identity without changing the underlying route representation.

## V1 Scope

Hostcheck V1 currently collects:

* IPv4 routes
* IPv6 routes
* destination addresses when provided
* destination prefix lengths
* source addresses when provided
* source prefix lengths
* gateways when provided
* output interface indexes
* route priority
* routing table identifiers
* route protocol
* route scope
* route type
* route flags
* multipath next hops
* interface identity and state
* interface addresses
* network observation timestamp

Hostcheck V1 does not currently:

* reimplement Linux route selection
* collect the complete routing-rule system
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
* determine whether a configured route is actually reachable
* interpret multipath selection behavior
* treat the network observation as an atomic transaction

Route interface indexes are intentionally preserved as kernel indexes in the `Route` model. Their relationship to interface names is established by the higher-level `Network` snapshot rather than by changing the low-level route representation.

## Lab Environment Limitations

The current experiments were performed on a simple VirtualBox Linux host with:

* one network namespace
* one Ethernet interface
* one loopback interface
* IPv4 networking
* IPv6 networking
* simple main and local routing tables
* DHCP-provided IPv4 configuration
* router-advertisement-provided IPv6 configuration

More complex Linux systems may contain:

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
* multipath routes
* custom routing tables
* complex routing rules

These are outside the current V1 scope.

The current route model is therefore intentionally smaller than the complete Linux routing subsystem.

## Testing

The route collector has tests covering basic invariants and expected lab state.

The tests verify that:

* route collection succeeds
* at least one route is returned
* route families are IPv4 or IPv6
* destination prefix lengths are valid
* source prefix lengths are valid
* destination addresses match the route family
* gateway addresses match the route family
* both IPv4 and IPv6 default routes are present
* the expected `10.0.2.0/24` IPv4 route is present
* multipath fields preserve the underlying interface index, gateway, hop value, and flags

The network snapshot tests additionally verify that:

* network collection succeeds
* an observation timestamp is recorded
* at least one interface is returned
* at least one route is returned
* collected interface indexes are valid
* collected interface addresses are non-nil
* route interface indexes can be correlated with collected interface identity

The default route tests rely on the current rtnetlink representation:

```text
PrefixLen   = 0

Destination = nil
```

for a default route.

The tests are intentionally based partly on the controlled lab environment.

As Hostcheck becomes portable across different Linux hosts, environment-specific expectations may need to be separated from invariant route-model tests.

## Important Engineering Findings

### Finding 1: Routing is destination-dependent

Different destinations can select different routes on the same host and interface.

The `ip route get` experiments demonstrated this directly.

### Finding 2: Direct routes do not require gateways

A destination matching a directly connected route can be reached through the output interface without a gateway.

The lab's:

```text
10.0.2.0/24
```

route demonstrates this.

### Finding 3: IPv6 gateways can be link-local

The lab host uses:

```text
fe80::2
```

as its IPv6 default gateway.

A route model must therefore support IPv6 gateways without assuming that gateways are IPv4 addresses.

### Finding 4: Route state contains more than ordinary `ip route` output

Rtnetlink exposed local and broadcast routes that were not shown by the normal simplified route listing.

This demonstrates why the underlying kernel representation matters.

### Finding 5: Address classification is not route scope

Go's `net.IP` classification functions do not provide Linux route or address scope.

Linux-provided scope should therefore be preserved where scope semantics matter.

### Finding 6: Route configuration does not prove connectivity

A route is evidence of configured kernel state, not proof that packets can successfully reach a destination.

### Finding 7: Linux already owns route selection

Hostcheck should observe and report the kernel's routing state rather than implement another routing algorithm.

### Finding 8: `/proc` route files are useful for learning but not the production boundary

The `/proc/net/route` experiment helped expose the raw IPv4 route representation.

The production collector uses rtnetlink because it provides structured kernel networking messages.

### Finding 9: Interface names and kernel indexes are different identifiers

Human-facing tools use names such as:

```text
enp0s3
```

while rtnetlink route messages identify the output interface using:

```text
2
```

The low-level Hostcheck route model preserves the kernel interface index.

The higher-level network snapshot provides the interface collection needed to correlate that index with an interface name.

### Finding 10: Missing route attributes are meaningful

A route without a gateway is not necessarily incomplete.

A directly connected route legitimately has no gateway.

Hostcheck therefore preserves absent attributes instead of manufacturing values to make every route look structurally identical.

### Finding 11: Multipath requires explicit modeling

A route can contain multiple next hops.

Flattening such a route into a single gateway would lose information.

Hostcheck therefore preserves multipath routes as a collection of `NextHop` values rather than flattening them into the primary gateway.

The collector preserves the raw next-hop fields without attempting to reproduce Linux's multipath selection behavior.

### Finding 12: A network observation is not an atomic kernel snapshot

The network snapshot combines information from multiple kernel interfaces.

Interface configuration, addresses, routes, and runtime counters can change independently during collection.

`Network.ObservedAt` records the beginning of the observation cycle and provides temporal context, but it does not imply that every field was read simultaneously.

This is intentional. Hostcheck reports the state it observed during a collection cycle rather than manufacturing transactional semantics that Linux does not provide across these interfaces.

## Current Architecture

The networking subsystem is currently structured as:

```text
internal/network/

    address.go
    address_reader.go

    enumerate.go

    interface.go
    interface_reader.go

    network.go

    route.go
    route_reader.go

    statistics.go
    statistics_reader.go
```

The data models are separated from the collection logic.

The higher-level network collector assembles:

```text
interface identity/state
        +
interface addresses
        +
routing state
        |
        v
     Network
```

The routing collector depends on:

```text
Go standard library

    net.IP

Linux rtnetlink

    route list

Linux interface indexes

    output interface identity
```

The collector does not execute:

```bash
ip route
```

and does not parse:

```text
/proc/net/route
```

for production collection.

The `/proc` route work remains an experiment used to understand Linux behavior.

## Current V1 Boundary

The current networking work establishes a useful boundary:

```text
Linux kernel networking state

            |

            v

     Linux interfaces

            |

            v

     Hostcheck collectors

            |

            v

      Hostcheck models

            |

            v

     Network observation

            |

            v

     future presentation
```

The collector should preserve meaningful kernel state first.

Human-readable representations such as:

```text
protocol=dhcp

scope=link

type=local

interface=enp0s3
```

can be produced later by a presentation layer.

The core collectors should not discard the underlying values merely because the final output may use human-readable names.

The higher-level `Network` model provides correlation between independently collected interface and route state while retaining the raw kernel identifiers in the underlying models.

## Conclusion

The routing experiments established that Linux routing is a kernel-managed system involving multiple address families, routing tables, routing rules, route types, priorities, protocols, scopes, gateways, source information, and output interfaces.

Hostcheck V1 therefore uses structured Linux networking information through rtnetlink rather than parsing command output or using legacy `/proc` route files as its production collection mechanism.

The current collector preserves meaningful kernel route state including destination and source prefixes, gateways, output interface indexes, priorities, tables, protocols, scopes, route types, flags, and multipath next hops.

The low-level collectors remain deliberately close to the kernel representation. Interface indexes are preserved rather than prematurely converted into names, while the higher-level network snapshot provides the context required to correlate route interfaces with collected interface identity.

The network snapshot is a best-effort observation assembled from multiple Linux kernel interfaces. `Network.ObservedAt` records the beginning of the collection cycle and provides temporal context, but the snapshot is not an atomic transaction.

The collector does not attempt to reproduce Linux route selection, policy routing, connectivity testing, or packet forwarding.

Those responsibilities remain with the Linux kernel.

Further networking work should be deliberate scope expansion based on an actual Hostcheck requirement rather than additional collection for its own sake.
