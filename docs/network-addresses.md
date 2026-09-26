# Network Addresses

## Purpose

Hostcheck needs to report the addresses configured on each network interface while preserving information that is meaningful to Linux networking.

An interface can have multiple addresses and multiple address families. The collector therefore represents addresses as a collection rather than reducing an interface to a single IP address.

## Linux Address Model

A network interface can have:

* zero or more IPv4 addresses
* zero or more IPv6 addresses
* different prefix lengths
* different address scopes
* multiple addresses from the same address family

For example, the lab host has the following addresses:

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

## Address Collection

Hostcheck currently uses Go's standard networking API:

```go
net.InterfaceByName
net.Interface.Addrs
```

The collection flow is:

```text
interface name
      |
      v
net.InterfaceByName
      |
      v
net.Interface.Addrs
      |
      v
*net.IPNet
      |
      +---- IP
      |
      +---- prefix length
      |
      v
Hostcheck Address
```

The collector expects the returned address representation to be `*net.IPNet`.

The prefix length is obtained from the network mask using:

```go
prefixLen, bits := ipnet.Mask.Size()
```

The collector validates that the address width is either:

```text
32 bits
128 bits
```

Anything else is rejected.

## Lab Experiment

The address inspection experiment used:

```bash
ip -br addr
```

and Go's networking API.

The host reported:

```text
lo       UNKNOWN 127.0.0.1/8 ::1/128

enp0s3   UP
         10.0.2.15/24
         fd17:625c:f037:2:a00:27ff:fed0:dc4d/64
         fe80::a00:27ff:fed0:dc4d/64
```

Go's `net.Interface.Addrs()` produced:

```text
interface=lo
  type=*net.IPNet network=ip+net string=127.0.0.1/8
  ip=127.0.0.1 mask=ff000000 prefix=8 bits=32

  type=*net.IPNet network=ip+net string=::1/128
  ip=::1 mask=ffffffffffffffffffffffffffffffff prefix=128 bits=128

interface=enp0s3
  type=*net.IPNet network=ip+net string=10.0.2.15/24
  ip=10.0.2.15 mask=ffffff00 prefix=24 bits=32

  type=*net.IPNet network=ip+net string=fd17:625c:f037:2:a00:27ff:fed0:dc4d/64
  ip=fd17:625c:f037:2:a00:27ff:fed0:dc4d
  mask=ffffffffffffffff0000000000000000
  prefix=64
  bits=128

  type=*net.IPNet network=ip+net string=fe80::a00:27ff:fed0:dc4d/64
  ip=fe80::a00:27ff:fed0:dc4d
  mask=ffffffffffffffff0000000000000000
  prefix=64
  bits=128
```

## Address Scope

Linux assigns scope information to addresses.

For example, the lab host reports:

```text
127.0.0.1/8
    scope host

10.0.2.15/24
    scope global

fd17:625c:f037:2:a00:27ff:fed0:dc4d/64
    scope global

fe80::a00:27ff:fed0:dc4d/64
    scope link

::1/128
    scope host
```

Address scope is a Linux networking concept and should not be inferred from general-purpose IP classification functions.

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

In particular:

```text
fd17:625c:f037:2:a00:27ff:fed0:dc4d
```

was classified by Go as both private and global-unicast.

Linux, however, reports the address with:

```text
scope global
```

Similarly, an IPv6 link-local address is identified by Go using IP classification functions, but this is not equivalent to obtaining Linux's actual address scope metadata.

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

If Linux scope becomes a required Hostcheck field, it should be collected from a Linux-native source such as netlink rather than inferred.

## `ip` Output Experiment

The `ip` command was also inspected using:

```bash
ip -o addr
```

The output contains multiple fields whose positions depend on the particular address and its associated metadata.

An initial assumption that a fixed field position contained the address scope was incorrect.

For an IPv4 address, the broadcast field appeared before the scope field:

```text
... brd 10.0.2.255 scope global ...
```

Therefore, treating a fixed positional field as the scope would incorrectly return:

```text
10.0.2.255
```

instead of:

```text
global
```

### Design consequence

The `ip` command is useful for human inspection and validation, but Hostcheck should not parse its textual output for production collection.

## Why `net.Interface.Addrs()` Is Currently Appropriate

The current address requirements are limited to:

* IP address
* prefix length
* IPv4/IPv6 distinction

Go's standard library provides these without requiring command execution or a third-party dependency.

This makes it an appropriate implementation for the current V1 address collector.

However, the standard API does not expose all Linux-specific address metadata.

In particular, Linux address scope is not currently represented by the Hostcheck model.

If future requirements require Linux-specific address attributes, the collector can move to a Linux-native interface such as netlink.

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

## Address Lifetime and Dynamic Addresses

The lab host's addresses include dynamic configuration metadata when viewed through `ip`.

For example, the IPv4 address is marked:

```text
dynamic
```

and has valid and preferred lifetimes.

The current Hostcheck V1 model does not expose address lifetimes or dynamic/static state.

These are candidates for future Linux-specific address metadata collection if they become relevant to host reliability.

## V1 Scope

Hostcheck V1 collects:

* interface-associated IP addresses
* IPv4 and IPv6 addresses
* prefix lengths

Hostcheck V1 does not currently collect:

* Linux address scope
* address labels
* address lifetimes
* preferred lifetimes
* dynamic/static state
* broadcast address
* deprecated state
* tentative state
* duplicate-address-detection state
* routing information derived from addresses

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

The textual output is intended for users and is not a stable application interface.

### Finding 6: Standard library is sufficient for the current V1 model

The current requirements do not justify a Linux-specific dependency solely for basic address and prefix collection.

## Limitations

The current collector intentionally exposes a small cross-platform representation rather than every Linux-specific address attribute.

This means some information visible through:

```bash
ip -j addr
```

is not currently represented.

That is deliberate V1 scope rather than missing data caused by an implementation failure.

## Conclusion

Network addresses are modeled as a collection of IP/prefix pairs associated with each interface.

Hostcheck currently uses Go's standard networking API because it provides the required address and prefix information without parsing external command output.

Linux-specific metadata such as address scope is intentionally not inferred. If such metadata becomes necessary, it should be collected from a Linux-native interface rather than approximated from generic IP classification.
