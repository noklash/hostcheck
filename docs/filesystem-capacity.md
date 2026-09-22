```markdown
# Filesystem Capacity and Inode Accounting

## Purpose

The filesystem subsystem reads Linux filesystem capacity and inode statistics directly from the kernel and converts them into a small host-level model.

The goal is to understand what Linux actually reports before building higher-level health or monitoring logic around it.

V1 is path-scoped. The caller provides a path such as `/` or `/tmp`, and the subsystem reports statistics for the filesystem containing that path.

## Linux source

The subsystem uses the Linux `statfs` interface.

At the Go level, V1 calls:

```go
syscall.Statfs(path, &stat)
````

The kernel returns a `struct statfs`-compatible structure containing filesystem type, block information, inode information, and other metadata.

The implementation currently uses Go's `syscall` package because this project is Linux-specific and the purpose of V1 is to learn and expose the underlying Linux interface directly.

`syscall` is frozen in modern Go and is not the preferred package for new portable APIs. This is therefore a deliberate V1 implementation choice rather than a general recommendation for new Go software.

## Kernel fields used

The collector maps the following `statfs` fields into the hostcheck model:

| Linux field | Go field | Meaning                                                  |
| ----------- | -------- | -------------------------------------------------------- |
| `f_bsize`   | `Bsize`  | Filesystem block size reported by `statfs`               |
| `f_blocks`  | `Blocks` | Total filesystem blocks                                  |
| `f_bfree`   | `Bfree`  | Free blocks, including blocks reserved by the filesystem |
| `f_bavail`  | `Bavail` | Blocks available to unprivileged processes               |
| `f_files`   | `Files`  | Total inode count                                        |
| `f_ffree`   | `Ffree`  | Free inode count                                         |

The V1 model intentionally keeps these raw counts instead of immediately converting everything into human-readable units.

## Data model

The filesystem collector exposes:

```go
type Stats struct {
    Path            string
    BlockSize       uint64
    BlocksTotal     uint64
    BlocksFree      uint64
    BlocksAvailable uint64
    InodesTotal     uint64
    InodesFree      uint64
}
```

The distinction between free and available blocks is preserved because they have different meanings.

## Free blocks versus available blocks

`f_bfree` represents blocks that are free from the filesystem's perspective.

`f_bavail` represents blocks available to an unprivileged process.

These values can differ because filesystems can reserve part of their capacity for privileged use or filesystem operation.

On the test filesystem containing `/`, the observed values were approximately:

```text
Total blocks:       10,336,806
Free blocks:         6,036,xxx
Available blocks:    5,503,xxx
```

The difference was roughly 533,000 blocks, or about 2 GiB at a 4096-byte block size.

This is why the collector does not replace `BlocksFree` with `BlocksAvailable`.

## Inodes are a separate resource

Filesystem capacity and inode capacity measure different resources.

Blocks represent filesystem storage capacity.

Inodes represent filesystem objects such as regular files and directories.

A filesystem can therefore have substantial free byte capacity while approaching its inode limit.

The V1 model keeps both resources independently:

```text
BlocksTotal
BlocksFree
BlocksAvailable

InodesTotal
InodesFree
```

Derived inode usage is:

```text
UsedInodes = InodesTotal - InodesFree
```

Available inode percentage is:

```text
AvailableInodePercent =
    InodesFree / InodesTotal * 100
```

## Capacity calculations

The collector exposes several derived calculations.

### Used blocks

```text
UsedBlocks = BlocksTotal - BlocksFree
```

This represents blocks that are not currently free according to `statfs`.

It should not be interpreted as exact user-data consumption because filesystem-reserved space and other filesystem-level accounting can contribute to the difference.

### Available capacity

```text
AvailableBytes = BlocksAvailable * BlockSize
```

This represents capacity available to an unprivileged process according to the `statfs` values.

### Used capacity

```text
UsedBytes = UsedBlocks * BlockSize
```

This is the byte representation of the non-free blocks.

It is deliberately kept distinct from `AvailableBytes`.

### Available capacity percentage

```text
AvailablePercent =
    BlocksAvailable / BlocksTotal * 100
```

This measures the percentage of total blocks available to an unprivileged process.

It is not the same quantity as the `Use%` displayed by `df`.

## Units

The raw block counters returned by `statfs` are counts of filesystem blocks.

V1 multiplies block counts by the reported `Bsize` when converting them to bytes.

The observed filesystems in the development environment reported:

```text
Block size:       4096 bytes
Fundamental size: 4096 bytes
```

The implementation currently models `Bsize` and does not expose `Frsize`.

This is a deliberate V1 simplification. If filesystem types with different `f_bsize` and `f_frsize` semantics become relevant, the byte-accounting model should be revisited rather than assuming the current conversion is universally sufficient.

## Path-scoped collection

The collector accepts an explicit path:

```go
ReadStats("/")
ReadStats("/tmp")
```

This matters because different paths can belong to different filesystems.

For example, during development:

```text
/      -> /dev/sda2
/tmp   -> tmpfs
```

The same host therefore exposed different capacity and inode statistics depending on the path queried.

V1 does not discover every mounted filesystem.

Mount enumeration, mount classification, duplicate filesystem handling, and policy for which filesystems should be monitored are deferred until the host-level architecture requires them.

## Controlled experiments

The filesystem implementation was validated against both live kernel state and controlled experiments.

### Experiment 1: `/` versus `/tmp`

The collector was run against both paths.

For `/`:

```text
Path: /
Block size: 4096 bytes
Total blocks: 10336806
Free blocks: 6036953
Available blocks: 5503821
Used blocks: 4299853
Available capacity: 22543650816 bytes
Used capacity: 17612197888 bytes
Available capacity: 53.24%
Total inodes: 2646016
Free inodes: 2355860
Used inodes: 290156
Available inodes: 89.03%
```

For `/tmp`:

```text
Path: /tmp
Block size: 4096 bytes
Total blocks: 435046
Free blocks: 434932
Available blocks: 434932
Used blocks: 114
Available capacity: 1781481472 bytes
Used capacity: 466944 bytes
Available capacity: 99.97%
Total inodes: 1048576
Free inodes: 1048478
Used inodes: 98
Available inodes: 99.99%
```

The experiment demonstrated that path selection determines which filesystem is inspected.

### Experiment 2: Creating 10,000 empty files

A controlled experiment created 10,000 empty files on `/tmp`.

Before:

```text
IUsed = 98
IFree = 1048478
```

After:

```text
IUsed = 10098
IFree = 1038478
```

The filesystem therefore consumed exactly 10,000 additional inodes.

The human-readable `df -h` output did not show a meaningful byte-usage change because the files contained no data and the displayed values were rounded.

This demonstrates why byte capacity and inode capacity must be monitored independently.

### Experiment 3: Writing 10 MiB

A 10 MiB file was written to `/tmp`.

The command used was:

```bash
dd if=/dev/zero of=/tmp/hostcheck-fs-test/data.bin bs=1M count=10 status=progress
```

The filesystem changed from approximately:

```text
Used = 456K
IUsed = 98
```

to:

```text
Used = 11M
IUsed = 99
```

The experiment demonstrated two separate effects:

1. Creating the file consumed an inode.
2. Writing 10 MiB consumed filesystem capacity.

The two resources are related to the same filesystem object but are accounted for independently by the filesystem.

## Live-state behavior

`statfs` reports live filesystem state.

The values are not a transactional snapshot of all filesystem activity.

Filesystem state can change between:

* reading the statistics,
* calculating derived values,
* displaying the results.

Concurrent file creation, deletion, writes, filesystem activity, and other processes can therefore cause small differences between successive observations.

The collector does not attempt to provide transactional consistency.

## Validation strategy

The filesystem subsystem uses two forms of testing.

### Live integration test

`TestReadStats` calls the real Linux `statfs` interface against `/`.

This verifies that the collector works against the running Linux environment.

### Controlled collector tests

The collector also accepts an injected `statfs` function internally.

This allows tests to provide deterministic `syscall.Statfs_t` values and verify:

* field mapping,
* path forwarding,
* error propagation,
* invalid block-size handling.

The tests therefore do not depend entirely on the current filesystem state.

### Derived calculation tests

Capacity and inode calculations are tested independently.

The tests cover:

* used blocks,
* invalid free-block counts,
* available bytes,
* zero block size,
* used bytes,
* available capacity percentage,
* used inodes,
* available inode percentage.

Arithmetic overflow is explicitly checked when converting block counts to bytes.

## Error handling

The collector rejects an invalid non-positive block size before converting it to `uint64`.

Derived calculations reject invalid states such as:

```text
Free blocks > total blocks

Free inodes > total inodes

Total blocks = 0

Total inodes = 0

Block size = 0
```

The byte conversion also checks for `uint64` multiplication overflow.

## V1 scope

The current filesystem subsystem intentionally focuses on:

* filesystem capacity,
* free blocks,
* available blocks,
* inode capacity,
* free inodes,
* derived capacity and inode percentages,
* explicit path-based collection,
* direct Linux `statfs` integration.

The following are intentionally deferred:

* discovering all mounted filesystems,
* filesystem mount classification,
* filesystem type policy,
* per-mount host snapshots,
* filesystem latency,
* I/O throughput,
* I/O error accounting,
* Prometheus exposition,
* alerting,
* daemon behavior.

Those concerns belong to later layers of the hostcheck architecture.

## Engineering conclusion

The important result of this subsystem is not the `df`-like output.

The important result is the accounting model underneath it:

```text
filesystem
    |
    +-- blocks
    |     +-- total
    |     +-- free
    |     +-- available to unprivileged processes
    |
    +-- inodes
          +-- total
          +-- free
```

Capacity exhaustion and inode exhaustion are separate failure modes.

The collector preserves the kernel's distinctions and leaves health policy for a later layer.
```