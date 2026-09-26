# Network Address Experiments

These experiments investigate how Linux and Go represent addresses assigned to network interfaces.

The experiments were performed before and during implementation of Hostcheck's network address collector.

## 1. Inspect Interface Addresses

Command:

```bash
ip -br addr
```

Observed:

```text
lo       UNKNOWN 127.0.0.1/8 ::1/128

enp0s3   UP
         10.0.2.15/24
         fd17:625c:f037:2:a00:27ff:fed0:dc4d/64
         fe80::a00:27ff:fed0:dc4d/64
```

The interface `enp0s3` has:

* one IPv4 address
* two IPv6 addresses

The loopback interface has:

* one IPv4 address
* one IPv6 address

This immediately establishes that an interface cannot be represented by a single IP address.

## 2. Inspect Addresses Through Go

The experiment used Go's:

```go
net.InterfaceByName
net.Interface.Addrs
```

The observed address objects were:

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

The returned values are `*net.IPNet`.

The prefix length can be obtained from the mask using:

```go
prefixLen, bits := ipnet.Mask.Size()
```

## 3. Address Width

The experiment confirmed two relevant address widths:

```text
IPv4 = 32 bits
IPv6 = 128 bits
```

Hostcheck validates that the address mask reports one of these widths.

Unexpected address widths are rejected rather than silently accepted.

## 4. Address Classification Experiment

A separate experiment tested Go's IP classification functions.

Representative output:

```text
127.0.0.1 -> loopback=true private=false linklocal=false globalunicast=false
::1 -> loopback=true private=false linklocal=false globalunicast=false

10.0.2.15 -> loopback=false private=true linklocal=false globalunicast=true
172.16.1.10 -> private=true globalunicast=true
192.168.1.10 -> private=true globalunicast=true

8.8.8.8 -> globalunicast=true

fd17:... -> private=true globalunicast=true

fe80::... -> linklocal=true globalunicast=false

2001:4860:4860::8888 -> globalunicast=true
```

The purpose was to determine whether Go's generic IP classifications could be used to infer Linux address scope.

They cannot.

## 5. Linux Scope Comparison

The same host was inspected with:

```bash
ip -j addr
```

Linux reported:

```text
127.0.0.1/8
    scope host

::1/128
    scope host

10.0.2.15/24
    scope global

fd17:625c:f037:2:a00:27ff:fed0:dc4d/64
    scope global

fe80::a00:27ff:fed0:dc4d/64
    scope link
```

The important discrepancy was:

```text
Go:
fd17:... -> private + global unicast

Linux:
fd17:... -> scope global
```

This demonstrates that IP classification and Linux address scope are different concepts.

## 6. `ip -o addr` Parsing Experiment

The command:

```bash
ip -o addr
```

was inspected to understand whether its text format could safely be parsed.

An IPv4 address line contains information such as:

```text
... brd 10.0.2.255 scope global ...
```

A fixed-position parser can therefore mistake:

```text
10.0.2.255
```

for:

```text
global
```

if it assumes the scope field always occurs at the same position.

This experiment established that parsing `ip` command text using fixed field positions is brittle.

Hostcheck therefore does not use `ip` command output for production address collection.

## 7. Structured `ip` Output

The command:

```bash
ip -j addr
```

provides structured JSON.

For `enp0s3`, the relevant information included:

```json
{
  "ifindex": 2,
  "ifname": "enp0s3",
  "addr_info": [
    {
      "family": "inet",
      "local": "10.0.2.15",
      "prefixlen": 24,
      "scope": "global"
    },
    {
      "family": "inet6",
      "local": "fd17:625c:f037:2:a00:27ff:fed0:dc4d",
      "prefixlen": 64,
      "scope": "global"
    },
    {
      "family": "inet6",
      "local": "fe80::a00:27ff:fed0:dc4d",
      "prefixlen": 64,
      "scope": "link"
    }
  ]
}
```

This is useful as a validation reference.

Hostcheck does not execute `ip` and parse this JSON in production.

## 8. Current Hostcheck Model

The current model is intentionally small:

```go
type Address struct {
    IP        net.IP
    PrefixLen int
}
```

There is no separate:

```go
Family string
```

because IPv4 versus IPv6 can be derived from `net.IP`.

There is also no:

```go
Scope string
```

because Go's standard address API does not provide Linux address scope directly.

Adding a guessed scope would create misleading data.

## 9. Design Decision

The address collector uses:

```go
net.InterfaceByName
net.Interface.Addrs
```

rather than:

```text
ip addr
ip -o addr
ip -j addr
/proc files
```

The standard library provides the information required by the current V1 model without requiring external commands or additional dependencies.

If future requirements need Linux-specific address metadata, the implementation can move to a Linux-native networking interface such as netlink.

## 10. Implementation Validation

The address collector was validated against the lab host.

Expected loopback addresses:

```text
127.0.0.1/8
::1/128
```

Expected `enp0s3` addresses:

```text
10.0.2.15/24
fd17:625c:f037:2:a00:27ff:fed0:dc4d/64
fe80::a00:27ff:fed0:dc4d/64
```

The implementation preserves both the IP address and prefix length.

## 11. Important Findings

### Finding 1: An interface can have multiple addresses

Address data must therefore be represented as a collection.

### Finding 2: IPv4 and IPv6 can coexist

The same interface can carry addresses from both families.

### Finding 3: Prefix length is part of the configuration

The address and prefix must be preserved together.

### Finding 4: Generic IP classification does not equal Linux scope

Go's classification helpers cannot be treated as authoritative Linux scope information.

### Finding 5: `ip` text output is unsuitable for production parsing

Field positions can vary depending on address and route metadata.

### Finding 6: Structured kernel data is preferable

The experiment supports using structured APIs for production collection rather than parsing command output.

### Finding 7: Scope remains intentionally unresolved in the current model

Linux scope is known to be important, but the current standard-library collector does not expose it directly.

This is an explicit design limitation, not an accidental omission.

## 12. V1 Scope

The address experiment and collector cover:

* interface-associated IPv4 addresses
* interface-associated IPv6 addresses
* prefix lengths
* address-family derivation

They do not cover:

* Linux address scope collection
* address lifetime collection
* preferred lifetime
* dynamic/static state
* broadcast addresses
* address labels
* tentative/deprecated state
* duplicate-address detection
* route collection

Those concerns belong to separate networking subsystems or future extensions.

## 13. Conclusion

The address experiments established that an interface is a container for multiple address objects rather than a single IP.

They also demonstrated that Linux-specific address semantics cannot always be reconstructed from generic Go IP classification.

Hostcheck therefore keeps the current address model deliberately small and accurate:

```text
Address
├── IP
└── PrefixLen
```

Linux-specific metadata will only be added when it can be collected from an authoritative source.
