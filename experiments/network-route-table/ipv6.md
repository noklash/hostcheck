# Raw IPv6 Route Table

## Objective

Inspect the raw IPv6 routing representation exposed by Linux through `/proc/net/ipv6_route`.

This experiment is for understanding the kernel representation only. It is not a production parser for Hostcheck.

## Command

```bash
cat /proc/net/ipv6_route
```

## Observed Output

The current VM exposes entries such as:

```text
fd17625cf03700020000000000000000 40 00000000000000000000000000000000 00 00000000000000000000000000000000 00000064 00000003 00000000 00000001   enp0s3
fe800000000000000000000000000000 40 00000000000000000000000000000000 00 00000000000000000000000000000000 00000400 00000002 00000000 00000001   enp0s3
00000000000000000000000000000000 00 00000000000000000000000000000000 00 fe800000000000000000000000000002 00004e84 00000003 00000000 00000003   enp0s3
00000000000000000000000000000001 80 00000000000000000000000000000000 00 00000000000000000000000000000000 00000000 00000003 00000000 80200001       lo
```

The first field is the hexadecimal IPv6 destination prefix.

The second field is the prefix length.

Other fields encode the next hop, metric, flags, and interface-related routing information.

## What This Shows

The raw IPv6 representation is substantially less readable than the output produced by `ip -6 route`.

For example, the default route appears in the raw representation as a destination consisting entirely of zeroes with a prefix length of `00`, while the next hop is represented as the hexadecimal form of:

```text
fe80::2
```

The interface is still identified as:

```text
enp0s3
```

This corresponds to the higher-level route:

```text
default via fe80::2 dev enp0s3
```

## Why It Matters

The experiment demonstrates that Linux exposes routing state at a lower level than the representation normally used by administrators.

Understanding this representation is useful when investigating Linux networking, but Hostcheck should not depend on manually decoding `/proc/net/ipv6_route`.

The production network collector should use Linux-native rtnetlink/netlink interfaces.

## Scope

This experiment intentionally does not:

* implement a complete IPv6 route parser
* decode every field
* reproduce Linux route selection
* interpret routing flags exhaustively
* replace rtnetlink
* become part of the production collector

The purpose is to understand the underlying representation before selecting the production abstraction.
