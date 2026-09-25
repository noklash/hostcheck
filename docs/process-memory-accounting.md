# Linux Process Memory Accounting

## Purpose

This document explains the process memory statistics collected by hostcheck, where they come from, what they represent, and the limitations of interpreting them.

The goal is to understand the difference between a process's virtual address space and its resident physical memory well enough to expose useful process-level information without pretending that one number represents total memory consumption.

## Source

Hostcheck reads:

```text
/proc/<pid>/status
```

For process memory accounting, V1 uses these fields:

| Field      | Hostcheck field     | Meaning                                               | Unit |
| ---------- | ------------------- | ----------------------------------------------------- | ---- |
| `VmSize`   | `VirtualBytes`      | Total virtual memory size of the process              | kB   |
| `VmRSS`    | `ResidentBytes`     | Resident memory currently associated with the process | kB   |
| `RssAnon`  | `AnonymousBytes`    | Resident anonymous memory                             | kB   |
| `RssFile`  | `FileBackedBytes`   | Resident file-backed memory                           | kB   |
| `RssShmem` | `SharedMemoryBytes` | Resident shared-memory portion                        | kB   |

Linux reports these values in kB. Hostcheck converts them to bytes at the parsing boundary so the process memory model has one consistent internal unit.

## Virtual Memory Versus Resident Memory

`VmSize` and `VmRSS` answer different questions.

`VmSize` describes the process's virtual address space. It includes address space that may not currently correspond to resident physical pages.

`VmRSS` describes memory that is currently resident.

Therefore:

```text
VirtualBytes != physical memory currently resident
```

A process can have a large virtual address space while using comparatively little physical memory.

Virtual memory can include:

* executable mappings
* shared libraries
* anonymous mappings
* reserved address space
* memory that has not yet been faulted into physical RAM

A large `VmSize` value is therefore not, by itself, evidence of high physical memory pressure.

## Anonymous and File-Backed Memory

Hostcheck also separates resident memory into:

```text
RssAnon
RssFile
RssShmem
```

This gives more context than RSS alone.

Anonymous memory is memory that is not backed by a regular file, such as many heap and stack allocations.

File-backed memory includes resident pages associated with files and mapped objects such as executables and shared libraries.

Shared memory is reported separately by Linux through `RssShmem`.

These values describe categories of resident memory. They should not automatically be treated as independent amounts that can always be summed to obtain a process's unique physical memory consumption, especially when shared pages are involved.

## Kernel Threads

Process enumeration through `/proc` includes kernel threads.

Kernel threads are different from normal userspace processes because they do not have a userspace address space and therefore do not expose the normal process memory fields used by Hostcheck.

For example, a kernel thread may contain:

```text
Kthread: 1
```

without:

```text
VmSize
VmRSS
RssAnon
RssFile
RssShmem
```

Hostcheck represents this explicitly:

```text
Process
├── Stats
├── Kthread
└── Memory
```

For a userspace process:

```text
Kthread = false
Memory != nil
```

For a kernel thread:

```text
Kthread = true
Memory = nil
```

A missing memory value therefore means that the metric is not applicable or available for that process. Hostcheck does not replace it with zero.

## Why Memory Is Read Together

Hostcheck reads `/proc/<pid>/status` once for each process and obtains both:

```text
Kthread
```

and the selected memory fields.

This avoids making separate `/proc/<pid>/status` reads for kernel-thread classification and memory collection.

The resulting collection path is:

```text
/proc/<pid>/status
        |
        v
   ReadStatus()
        |
        v
   ParseStatus()
        |
        v
      Status
      /    \
 Kthread   Memory
```

The production process collector combines this status information with process accounting obtained from `/proc/<pid>/stat`.

## Unit Conversion

Linux reports the selected memory fields in kB.

Hostcheck converts them to bytes while parsing:

```text
Linux kB
   |
   v
parser
   |
   v
bytes
```

This keeps unit conversion at the interface boundary and prevents the rest of the process subsystem from having to remember that the original kernel representation uses kB.

## Controlled Experiment

A controlled experiment was used to distinguish virtual address space from resident memory.

The experiment creates a 100 MiB anonymous memory mapping.

The process is measured:

1. before creating the mapping,
2. after creating the mapping but before touching its pages,
3. after writing to every page.

The experiment uses the system page size reported by Go rather than assuming a fixed page size.

One observed run produced:

```text
== before mmap ==
virtual:     1198.54 MiB
resident:       2.99 MiB
anonymous:      1.29 MiB

== after mmap, before page touch ==
virtual:     1298.86 MiB
resident:       6.41 MiB
anonymous:      4.71 MiB

== after touching every page ==
virtual:     1299.11 MiB
resident:     107.35 MiB
anonymous:    105.65 MiB
```

The measured deltas were:

```text
mmap reservation:
  virtual:   +100.32 MiB
  resident:    +3.42 MiB
  anonymous:   +3.42 MiB

page touching:
  virtual:     +0.25 MiB
  resident:  +100.93 MiB
  anonymous: +100.93 MiB
```

The process continues running and the Go runtime performs other allocations between measurements, so the values are not expected to remain completely static.

The important observation is the relationship between the second and third measurements:

```text
virtual memory:
    approximately unchanged

resident memory:
    increases by approximately 100 MiB

anonymous resident memory:
    increases by approximately 100 MiB
```

This demonstrates that reserving virtual address space does not require all corresponding pages to be resident immediately.

When the process writes to the previously untouched pages, the pages become resident and anonymous RSS increases.

## Process Lifetime Races

`/proc` is a live kernel interface.

A PID can disappear between:

```text
PID enumeration
```

and:

```text
/proc/<pid>/stat
/proc/<pid>/status
```

being read.

Hostcheck treats `ENOENT` during process collection as a normal race and skips the process rather than treating the entire collection as failed.

This is important for a host-level collector because processes can exit at any time.

## Data Model

The V1 process model is:

```text
Process
├── Stats
│   ├── PID
│   ├── Comm
│   ├── State
│   ├── PPID
│   ├── UTime
│   ├── STime
│   ├── Priority
│   ├── Nice
│   ├── Threads
│   ├── StartTime
│   ├── VSize
│   └── RSSPages
│
├── Kthread
└── Memory
    ├── VirtualBytes
    ├── ResidentBytes
    ├── AnonymousBytes
    ├── FileBackedBytes
    └── SharedMemoryBytes
```

`Memory` is a pointer because memory accounting is not applicable to kernel threads.

## What V1 Does Not Claim

The current process memory collector does not attempt to provide:

* proportional set size (PSS)
* unique physical memory ownership
* detailed mapping-level analysis
* heap allocation analysis
* memory leak detection
* cgroup memory limits
* container memory accounting
* OOM prediction
* memory pressure analysis
* `smaps` or `smaps_rollup` analysis

These require different kernel interfaces and different semantics.

In particular, RSS should not be presented as an exact measure of the physical memory uniquely owned by a process because pages can be shared.

## Why V1 Uses `/proc/<pid>/status`

The process memory implementation intentionally uses the relatively small set of fields required for host-level process observation.

`/proc/<pid>/smaps` and related interfaces expose substantially more mapping-level detail, including proportional accounting, but that complexity is not required for the current one-shot host-health model.

The V1 collector therefore favors:

```text
small interface
+
clear semantics
+
stable internal model
```

over collecting every available memory statistic.

## Current V1 Position

The process memory collector answers these questions:

1. How large is the process's virtual address space?
2. How much memory is currently resident?
3. How much resident memory is anonymous?
4. How much resident memory is file-backed?
5. How much resident memory is shared memory?
6. Is the process a kernel thread for which userspace memory accounting does not apply?

Further interpretation belongs above the raw collector layer.

Health thresholds and operational policies should not be embedded in the process collector.
