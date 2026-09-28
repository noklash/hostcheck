# Network Addresses

## Purpose

Hostcheck reports the addresses configured on each network interface while preserving information that is meaningful to Linux networking.

An interface can have multiple addresses and multiple address families. The collector therefore represents addresses as a collection rather than reducing an interface to a single IP address.

The collector also preserves the Linux kernel's address scope because scope is part of the kernel's networking state and cannot reliably be reconstructed from generic IP classification functions.

## Linux Address Model

A network interface can have:

* zero or more IPv4 addresses
* zero or more IPv6 addresses
* different prefix lengths
* different address scopes
* multiple addresses from the same address family

For example, the lab host currently has:

```text
lo
    127.0.0.1/8
    ::1/128

enp0s3
    10.0.2.15/24
    fd17:625c:f037:2:a00:27ff:fed0:dc4d/64
    fe80::a00:27ff:fed0:dc4d/64
```

The address collection therefore returns a slice of addresses for each interface.

## Hostcheck Address Model

The current V1 model is:

```go
type Address struct {
    IP        net.IP
    PrefixLen int
    Scope     uint8
}
```

The model intentionally does not contain a separate `Family` field.

IPv4 versus IPv6 can be derived from `net.IP`, so storing both would create redundant state that could become contradictory.

The prefix length is preserved because the address alone does not describe the network boundary.

For example:

```text
10.0.2.15
```

does not communicate the same information as:

```text
10.0.2.15/24
```

The prefix length is therefore part of the collected state.

The model also preserves the Linux kernel address scope as its numeric value rather than immediately converting it into a human-readable string.

This keeps the collected state close to the kernel representation and avoids embedding an incomplete scope mapping into the core data model.

## Address Collection

Hostcheck initially used Go's standard networking API:

```go
net.InterfaceByName
net.Interface.Addrs
```

That implementation was sufficient for basic IP and prefix collection, but the investigation demonstrated that the standard API does not expose the Linux address scope required by the current model.

Hostcheck therefore moved address collection to Linux rtnetlink.

The current collection flow is:

```text
interface name
      |
      v
net.InterfaceByName
      |
      | resolve interface name -> kernel interface index
      v
rtnetlink
      |
      v
RTM_GETADDR / address list
      |
      v
filter messages by interface index
      |
      +---- IP address
      |
      +---- prefix length
      |
      +---- Linux address scope
      |
      v
Hostcheck Address
```

`net.InterfaceByName` is still used, but only to resolve the interface name to its kernel interface index.

The actual address state is collected through rtnetlink.

The collector:

1. resolves the requested interface name
2. opens an rtnetlink connection
3. requests the kernel's address list
4. filters messages belonging to the requested interface index
5. obtains the address from the rtnetlink attributes
6. preserves the prefix length
7. preserves the Linux address scope
8. returns the resulting `Address` values

The implementation prefers the `Address` attribute and falls back to the `Local` attribute when necessary.

Missing address attributes are not replaced with fabricated values.

## Why Rtnetlink

Linux exposes networking configuration through kernel interfaces rather than requiring applications to parse human-oriented command output.

Rtnetlink provides structured kernel networking messages containing fields such as:

```text
address
prefix length
interface index
scope
flags
broadcast
label
```

Hostcheck only retains the fields required by the current model.

This gives the collector access to Linux-specific information without depending on the textual output of the `ip` command.

The choice is deliberate.

Hostcheck is intended to become a Linux infrastructure inspection tool, so where Linux-specific kernel state is required, the collector should use the corresponding kernel interface rather than infer or scrape the state from presentation-oriented command output.

## Address Scope

Linux assigns scope information to addresses.

The lab host currently reports:

```text
127.0.0.1/8
    scope 254

::1/128
    scope 254

10.0.2.15/24
    scope 0

fd17:625c:f037:2:a00:27ff:fed0:dc4d/64
    scope 0

fe80::a00:27ff:fed0:dc4d/64
    scope 253
```

These correspond to the commonly observed Linux scope meanings:

```text
254    host
253    link
0      global/universe
```

Hostcheck preserves the numeric kernel value rather than storing the textual names.

This matters because the numeric value is the actual value received from the kernel, while the textual representation is a presentation decision.

## Important Scope Experiment

Go's IP classification functions were tested against representative addresses.

Examples included:

```text
127.0.0.1
::1
10.0.2.15
172.16.1.10
192.168.1.10
8.8.8.8
fd17:625c:f037:2:a00:27ff:fed0:dc4d
fe80::a00:27ff:fed0:dc4d
2001:4860:4860::8888
```

The experiment demonstrated that Go's classifications do not map directly onto Linux address scope.

For example:

```text
fd17:625c:f037:2:a00:27ff:fed0:dc4d
```

can be classified by Go using generic IP properties, but that classification does not represent the Linux kernel's address scope field.

Likewise, identifying an IPv6 address as link-local through an IP classification function is not the same operation as reading the address's Linux scope metadata from the kernel.

### Design consequence

Hostcheck must not do this:

```go
if ip.IsPrivate() {
    scope = "..."
}
```

or:

```go
if ip.IsLinkLocalUnicast() {
    scope = "..."
}
```

and present that value as Linux address scope.

If Linux scope is required, it should be collected from a Linux-native source.

Hostcheck now does this through rtnetlink.

## `ip` Output Experiment

The `ip` command was inspected during the initial address investigation:

```bash
ip -o addr
```

The output contains multiple fields whose presence and position depend on the address and its associated metadata.

An initial assumption that a fixed field position contained the address scope was incorrect.

For example, an IPv4 address can appear as:

```text
... brd 10.0.2.255 scope global ...
```

The broadcast address therefore appears before the scope field.

Treating a fixed positional field as the scope would incorrectly return:

```text
10.0.2.255
```

instead of:

```text
global
```

### Design consequence

The `ip` command is useful for human inspection and validation, but Hostcheck should not parse its textual output for production collection.

This is one of the reasons the collector was moved to rtnetlink.

## Rtnetlink Address Experiment

The rtnetlink experiment was used to inspect the structured address messages exposed by the Linux kernel.

The observed address messages included:

```text
family=IPv4 index=1 prefix=8 scope=254 flags=128
  address=127.0.0.1
  local=127.0.0.1
  label=lo

family=IPv4 index=2 prefix=24 scope=0 flags=0
  address=10.0.2.15
  local=10.0.2.15
  broadcast=10.0.2.255
  label=enp0s3

family=IPv6 index=1 prefix=128 scope=254 flags=128
  address=::1
  local=<none>

family=IPv6 index=2 prefix=64 scope=0 flags=0
  address=fd17:625c:f037:2:a00:27ff:fed0:dc4d

family=IPv6 index=2 prefix=64 scope=253 flags=128
  address=fe80::a00:27ff:fed0:dc4d
```

This experiment established that the kernel provides the scope directly alongside the address and prefix information.

It also demonstrated why rtnetlink is a better production boundary for Linux-specific address collection than parsing command output.

## Interface Identification

The public collector accepts an interface name such as:

```text
enp0s3
```

The Linux kernel's rtnetlink address messages identify interfaces by numeric index.

The collector therefore performs:

```text
enp0s3
   |
   v
net.InterfaceByName
   |
   v
interface index = 2
   |
   v
rtnetlink address messages
   |
   v
select messages where Index == 2
```

This prevents addresses belonging to another interface from being returned.

The interface index is not stored in the current `Address` model because the address collection API is already scoped to a specific interface.

## Relationship Between Addresses and Routes

An address and a route are separate pieces of kernel networking state.

For example:

```text
Address:

    10.0.2.15/24
```

and:

```text
Route:

    10.0.2.0/24
    dev enp0s3
```

are related, but Hostcheck should not reconstruct the route from the address.

Linux may install, modify, or omit routes depending on configuration.

Address collection therefore observes addresses directly, while route collection observes routes separately.

The route collector uses rtnetlink independently of the address collector.

## Address Lifetime and Dynamic Addresses

The lab host's addresses include additional configuration metadata when viewed through `ip`.

For example, the IPv4 address can be marked:

```text
dynamic
```

and can have valid and preferred lifetimes.

The current Hostcheck V1 model does not expose:

* valid lifetime
* preferred lifetime
* dynamic/static state

These remain candidates for future Linux-specific address metadata collection if they become relevant to host reliability.

The absence of these fields from the current model is deliberate.

## Address Flags

Rtnetlink also exposes address flags.

For example, the lab host produced flags for some addresses:

```text
127.0.0.1
    flags=128

fe80::a00:27ff:fed0:dc4d
    flags=128
```

The current V1 model does not expose these flags.

Address flags can encode additional kernel state such as properties associated with temporary, deprecated, tentative, or duplicate-address-detection states.

These should not be inferred from the address itself.

If address flags become important to Hostcheck's reliability model, they should be collected directly from rtnetlink.

## V1 Scope

Hostcheck V1 currently collects:

* interface-associated IP addresses
* IPv4 and IPv6 addresses
* prefix lengths
* Linux address scope

Hostcheck V1 does not currently collect:

* address labels
* address lifetimes
* preferred lifetimes
* dynamic/static state
* broadcast address
* address flags
* deprecated state as a dedicated field
* tentative state as a dedicated field
* duplicate-address-detection state as a dedicated field
* routing information derived from addresses

The distinction is important:

Linux may expose some of this information through rtnetlink, but information being available in the kernel does not automatically make it part of Hostcheck's V1 contract.

## Important Engineering Findings

### Finding 1: Interfaces can have multiple addresses

An interface must not be modeled with a single `IP` field.

### Finding 2: IPv4 and IPv6 coexist

The same interface can simultaneously carry IPv4 and IPv6 addresses.

### Finding 3: Prefix length is meaningful state

The prefix is part of the network configuration and must be preserved.

### Finding 4: Go IP classification is not Linux scope

Generic IP classification should not be used as a substitute for Linux-provided address scope.

### Finding 5: `ip` output should not be parsed

The textual output is intended for users and is not the appropriate production interface for structured kernel state collection.

### Finding 6: Linux-specific requirements justify a Linux-native collector

The original standard-library implementation was sufficient for basic address and prefix collection.

Once Linux address scope became part of the required model, the standard API was no longer sufficient.

The collector therefore moved to rtnetlink.

### Finding 7: Preserve kernel state before adding presentation logic

The current model stores the Linux scope value as a numeric field.

Human-readable names such as `host`, `link`, and `global` can be introduced at a presentation layer if needed.

### Finding 8: Interface names and kernel indices are different identifiers

Users and Hostcheck APIs work with interface names, while rtnetlink identifies interfaces by numeric kernel index.

The collector explicitly resolves the name to an index before filtering kernel messages.

## Testing

The address tests verify the collector against the current Linux host.

The tests cover:

```text
127.0.0.1/8
    scope 254

::1/128
    scope 254

10.0.2.15/24
    scope 0

fd17:625c:f037:2:a00:27ff:fed0:dc4d/64
    scope 0

fe80::a00:27ff:fed0:dc4d/64
    scope 253
```

The tests therefore verify more than simple address discovery.

They also verify that the Linux scope returned by rtnetlink is preserved correctly.

The tests are intentionally based on the lab host's expected network configuration. This is appropriate for the current infrastructure lab because the test environment is controlled.

As Hostcheck becomes portable across different Linux environments, these environment-dependent assumptions may need to move toward invariant tests or controlled test fixtures.

## Limitations

The current collector intentionally exposes a small representation of Linux address state.

Rtnetlink provides substantially more information than Hostcheck currently retains.

For example:

```text
flags
broadcast
label
cache information
lifetimes
```

may be available depending on the address and kernel state.

Not collecting these fields is deliberate V1 scope rather than an implementation failure.

The current collector also does not attempt to normalize Linux-specific scope values into a universal cross-platform abstraction.

Hostcheck is currently a Linux infrastructure project, so preserving the kernel representation is more useful than hiding it behind an abstraction that may lose information.

## Current Architecture

The address subsystem is currently structured as:

```text
internal/network/
    address.go
    address_reader.go
    address_reader_test.go
```

The data model is defined separately from the collection logic.

The collector depends on:

```text
Go standard library
    net.InterfaceByName
    net.IP

Linux rtnetlink
    address list
```

The collector does not execute:

```bash
ip addr
```

and does not parse command output.

## Conclusion

Network addresses are modeled as a collection of IP/prefix/scope values associated with each interface.

Hostcheck initially used Go's standard networking API because it provided the required address and prefix information without command execution or external parsing.

The investigation showed that Linux address scope is meaningful kernel state and cannot reliably be reconstructed from generic IP classification.

The collector therefore moved to rtnetlink.

The current design resolves the requested interface name to its kernel index, reads structured address messages from the Linux kernel, filters them by interface index, and preserves the address, prefix length, and Linux scope.

This keeps the production collector close to the actual Linux networking state while avoiding dependency on human-oriented command output.

Future Linux-specific address metadata can be added deliberately as Hostcheck's reliability requirements evolve.
