
``
# Linux Memory Accounting

## Purpose

This document explains the memory statistics collected by hostcheck, where they come from, what they mean, and how they are interpreted.

The goal is to understand the Linux memory model well enough to collect useful host-health information without reducing memory state to a misleading "used/free" number.

## Source

Hostcheck reads:

```text
/proc/meminfo
````

`/proc/meminfo` is a kernel-provided interface containing current memory statistics.

The values collected by hostcheck are:

| Field          | Meaning                                                                   | Unit |
| -------------- | ------------------------------------------------------------------------- | ---- |
| `MemTotal`     | Total physical memory available to the system                             | kB   |
| `MemAvailable` | Estimated memory available for starting new applications without swapping | kB   |
| `MemFree`      | Completely unused physical memory                                         | kB   |
| `Buffers`      | Memory used for kernel buffers                                            | kB   |
| `Cached`       | Memory used for filesystem/page cache                                     | kB   |
| `SwapTotal`    | Total configured swap space                                               | kB   |
| `SwapFree`     | Currently unused swap space                                               | kB   |

The collector preserves the units reported by Linux rather than converting them during collection.

## Why `MemFree` Is Not Enough

A Linux host can have relatively little `MemFree` while still having substantial memory available to applications.

Linux deliberately uses otherwise-unused RAM for caches and other reclaimable purposes. Therefore, a low `MemFree` value does not by itself indicate memory pressure.

`MemAvailable` is more useful for host-health assessment because it represents the kernel's estimate of how much memory can be made available to applications without resorting to swapping.

Hostcheck therefore collects both values, but `MemAvailable` is the more important operational signal.

## Why We Do Not Reconstruct `MemAvailable`

Hostcheck does not calculate:

```text
MemAvailable = MemFree + Buffers + Cached
```

That would be an oversimplification.

`MemAvailable` is calculated by the Linux kernel using information about reclaimable memory and memory reserves. Reconstructing it from a few visible fields would therefore produce a different and potentially misleading value.

The collector uses the kernel's `MemAvailable` value directly.

## Swap Accounting

Hostcheck collects:

```text
SwapTotal
SwapFree
```

Swap used is derived as:

```text
SwapUsed = SwapTotal - SwapFree
```

The collector does not interpret non-zero swap usage as an automatic failure condition.

Pages can remain in swap after memory pressure has decreased. Therefore, swap usage must be interpreted together with current memory availability and other host state.

## Data Model

The current V1 model is:

```text
MemInfo
├── Total
├── Available
├── Free
├── Buffers
├── Cached
├── SwapTotal
└── SwapFree
```

All values are stored as `uint64` and retain the `kB` unit reported by `/proc/meminfo`.

Derived values such as swap used and available percentage should be calculated separately from raw collection.

## Parser Design

The parser processes the text representation of `/proc/meminfo`.

For the fields required by V1, it validates:

1. The field is recognized.
2. The line contains the expected three fields.
3. The unit is `kB`.
4. The numeric value is a valid unsigned integer.
5. Required fields are present.
6. Required fields do not appear more than once.

Unknown fields are ignored.

This is intentional because `/proc/meminfo` contains many more fields than hostcheck currently needs, and the kernel interface can evolve.

## Collector Design

The collection path is:

```text
/proc/meminfo
      |
      v
ReadMemInfo()
      |
      v
readMemInfo(io.Reader)
      |
      v
ParseMemInfo()
      |
      v
MemInfo
```

`ReadMemInfo()` is the production entry point and opens `/proc/meminfo`.

The internal reader-based function separates filesystem access from parsing. Tests can therefore provide controlled input without depending on the live state of the host.

## Experiment

A controlled memory-pressure experiment was performed by allocating 512 MiB in a Python process:

```bash
python3 -c 'x = bytearray(512 * 1024 * 1024); input("Allocated 512 MiB. Press Enter to release...")'
```

Before the allocation, the system reported approximately:

```text
Mem:  3.3Gi used 2.2Gi free 146Mi buff/cache 1.2Gi available 1.2Gi
Swap: 3.8Gi used 1.3Gi free 2.5Gi
```

During the allocation:

```text
Mem:  3.3Gi used 2.4Gi free 204Mi buff/cache 915Mi available 969Mi
Swap: 3.8Gi used 1.6Gi free 2.3Gi
```

After the allocation was released:

```text
Mem:  3.3Gi used 1.9Gi free 632Mi buff/cache 944Mi available 1.4Gi
Swap: 3.8Gi used 1.6Gi free 2.3Gi
```

The experiment demonstrated several important properties:

* Allocating memory changed available memory and swap behavior.
* Linux reclaimed and adjusted cached memory as memory pressure changed.
* Memory reported as available changed substantially after the workload ended.
* Swap usage did not immediately return to its previous level after the allocation was released.

This is why host memory health cannot be reduced to a single `used` value.

## Limitations

The current memory implementation is intentionally small.

It does not yet:

* classify anonymous versus file-backed memory,
* report slab memory,
* distinguish active and inactive pages,
* calculate a complete memory-pressure model,
* inspect PSI memory pressure,
* expose cgroup memory limits,
* monitor memory continuously,
* define production health thresholds.

Those are separate engineering questions and should only be added when they are required by the host-health model.

## Current V1 Position

For the current one-shot hostcheck implementation, the important memory questions are:

1. How much physical memory exists?
2. How much does Linux consider available?
3. How much is completely free?
4. How much memory is associated with buffers and cache?
5. How much swap exists?
6. How much swap remains free?

The collector answers those questions directly from the kernel interface.

Further interpretation belongs above the raw collector layer.
``

