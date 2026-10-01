# hostcheck

A Linux host health and reliability agent written in Go.

hostcheck reads operating-system interfaces exposed by Linux and turns them into structured host information. The project is built from the kernel interfaces upward, with each subsystem researched, experimentally verified, implemented, tested, and documented before moving to the next.

The initial version is intentionally a one-shot collector. Continuous monitoring, exporters, dashboards, orchestration integrations, and other operational layers are outside the current scope.

## Objective

Build a small but technically defensible Linux host inspection tool that can answer basic reliability questions directly from the operating system.

The project is also an engineering study of the Linux interfaces behind those answers.

The guiding principle is:

> Understand the system first. Then build the tool.

## Current State

hostcheck currently collects information from five Linux subsystems:

* CPU accounting
* Memory
* Filesystem capacity and inode usage
* Processes
* Network interfaces, addresses, and routes

The implementation is still deliberately below the health-policy layer. The collectors expose observed operating-system state. Health decisions and thresholds will be added separately after the underlying models are reviewed.

### CPU

CPU accounting is collected from:

```text
/proc/stat
```

The implementation supports:

* parsing aggregate CPU accounting
* cumulative CPU counter deltas
* counter regression detection
* aggregate CPU utilization
* separate treatment of I/O wait and steal time
* preservation of guest and guest-nice counters
* direct collection from `/proc/stat`
* deterministic parser and collector tests

CPU utilization is calculated from counter deltas between samples rather than assuming that the requested sleep duration represents the exact sampling interval.

### Memory

Memory statistics are collected from:

```text
/proc/meminfo
```

The current V1 model includes:

* `MemTotal`
* `MemAvailable`
* `MemFree`
* `Buffers`
* `Cached`
* `SwapTotal`
* `SwapFree`

The implementation validates required fields, units, numeric values, duplicate fields, and missing fields.

Memory accounting is documented separately in:

```text
docs/memory-accounting.md
```

A controlled 512 MiB memory-pressure experiment was performed to understand how available memory, free memory, cache, and swap changed under workload.

### Filesystem

Filesystem capacity and inode statistics are collected using the Linux `statfs` interface.

The current model records:

* filesystem path
* block size
* total blocks
* free blocks
* blocks available to an unprivileged process
* total inodes
* free inodes

Derived values include:

* used blocks
* used bytes
* available bytes
* available capacity percentage
* used inodes
* available inode percentage

The distinction between free blocks and blocks available to an unprivileged process is preserved because they represent different Linux filesystem semantics.

Filesystem accounting is documented separately in:

```text
docs/filesystem-capacity.md
```

### Processes

Process observations are collected from the live `/proc` filesystem.

The process layer currently supports:

* process enumeration
* process `/proc/<pid>/stat` parsing
* process lifecycle race handling
* process CPU accounting
* process CPU utilization
* process memory accounting from `/proc/<pid>/status`
* kernel-thread identification

Processes can disappear between enumeration and individual `/proc` reads. hostcheck treats this as an expected property of the live `/proc` filesystem and skips processes that disappear during collection.

Kernel threads are included in the process collection. Their `Kthread` field is set to `true`, and their userspace memory information is unavailable because kernel threads do not have the same userspace memory accounting exposed through `/proc/<pid>/status`.

Process memory accounting is documented separately in:

```text
docs/process-memory-accounting.md
```

Controlled experiments are used to verify how process CPU and memory measurements behave under known workloads.

### Network

Network state is collected from Linux networking interfaces and kernel routing information.

The current network snapshot includes:

* interface identity
* interface index
* interface name
* hardware address
* operational state
* carrier state
* MTU
* optional link speed
* optional duplex information
* interface addresses
* routing information
* multipath route next hops

Interface addresses and routes are collected through Linux rtnetlink rather than relying solely on high-level abstractions.

Routes retain kernel-level attributes including:

* address family
* destination and prefix length
* source and source prefix length
* gateway
* output interface index
* priority
* routing table
* protocol
* scope
* route type
* route flags
* multipath next hops

The network snapshot records an `ObservedAt` timestamp.

The snapshot is a best-effort observation of a live system rather than an atomic transaction. Interfaces, addresses, and routes can change while collection is taking place.

Network architecture and routing semantics are documented separately in:

```text
docs/network-addresses.md
docs/network-routing.md
```

## Usage

hostcheck is currently being developed as a one-shot Linux host inspection tool. The project is still focused on building and validating the underlying collectors, so usage currently consists of running the subsystem collectors and experiments directly.

### Requirements

hostcheck currently requires:

* Linux
* Go
* access to the Linux interfaces used by the collectors

The project is developed and tested on Linux because several collectors depend directly on Linux-specific interfaces such as `/proc`, `statfs`, and rtnetlink.

Check the installed Go version with:

```bash
go version
```

### Clone the Repository

Clone the repository and enter the project directory:

```bash
git clone https://github.com/noklash/hostcheck.git
cd hostcheck
```

### Run the Tests

Before using or modifying the project, run the complete test suite:

```bash
go test ./...
```

Run static analysis:

```bash
go vet ./...
```

Format the source tree:

```bash
gofmt -w .
```

Check for whitespace errors:

```bash
git diff --check
```

### Build the Project

Build all packages with:

```bash
go build ./...
```

At the current stage, hostcheck does not yet provide a single production command that assembles every collector into a complete host snapshot.

The repository is currently focused on developing and validating the subsystem collectors first.

### Run the Experiments

The repository contains small programs that can be used to inspect Linux behavior directly.

Run an experiment with:

```bash
go run ./experiments/<experiment>
```

For example:

```bash
go run ./experiments/cpu
```

Other available experiments include:

```text
experiments/cpu
experiments/filesystem
experiments/network-addresses
experiments/network-route-table
experiments/network-routing
experiments/network-rtnetlink
experiments/process
experiments/process-cpu
experiments/process-memory
```

Some experiments have their own README files containing additional instructions and observations.

For example:

```bash
cat experiments/cpu/README.md
cat experiments/process/README.md
cat experiments/network-routing/README.md
```

### Inspect Linux Interfaces Directly

The collectors are intentionally based on the same operating-system interfaces that can be inspected from the shell.

CPU accounting:

```bash
cat /proc/stat
```

Memory accounting:

```bash
cat /proc/meminfo
```

Process information:

```bash
cat /proc/self/stat
cat /proc/self/status
```

Filesystem statistics:

```bash
df -h /
df -i /
```

Network interfaces:

```bash
ip link
```

Network addresses:

```bash
ip addr
```

Routing information:

```bash
ip route
ip -6 route
```

These commands are useful when investigating collector behavior because they provide a human-readable view of the same underlying operating-system state that hostcheck is learning to collect programmatically.

### Using the Collectors

The current implementation is organized into internal subsystem packages.

The main packages are:

```text
internal/cpu
internal/memory
internal/filesystem
internal/process
internal/network
```

The collectors can be exercised through Go tests and the experiment programs while the host-level application layer is being developed.

The current design intentionally keeps collection separate from interpretation:

```text
Linux interface
      |
      v
Collector
      |
      v
Typed subsystem model
```

For example:

```text
/proc/stat
    |
    v
CPU collector
    |
    v
cpu.Stat
```

and:

```text
/proc/meminfo
    |
    v
Memory collector
    |
    v
memory.MemInfo
```

The same pattern is used for filesystem, process, and network information.

### Running Individual Tests

Run tests for a specific subsystem when working on that collector.

CPU:

```bash
go test ./internal/cpu
```

Memory:

```bash
go test ./internal/memory
```

Filesystem:

```bash
go test ./internal/filesystem
```

Processes:

```bash
go test ./internal/process
```

Network:

```bash
go test ./internal/network
```

Run a specific test with:

```bash
go test ./internal/cpu -run TestName
```

Replace `TestName` with the test you want to execute.

Verbose test output can be enabled with:

```bash
go test -v ./internal/cpu
```

### Running the Full Validation

The normal development validation sequence is:

```bash
gofmt -w .
go test ./...
go vet ./...
git diff --check
```

If all commands succeed, inspect the working tree:

```bash
git status --short --branch
```

### Current CLI Status

There is currently no stable command such as:

```bash
hostcheck
```

or:

```bash
go run .
```

that produces a complete host health report.

That is intentional.

The project is currently building the system from the bottom up:

```text
Linux interfaces
      |
      v
Subsystem collectors
      |
      v
Typed observations
      |
      v
Host snapshot
      |
      v
Health evaluation
      |
      v
Structured output
```

The host-level command will be introduced after the host snapshot and health-evaluation layers have been designed.

This prevents the command-line interface from becoming an accidental architectural boundary before the underlying host model is stable.

### Permissions and Failure Behavior

Most current collectors read information available to ordinary Linux users.

However, access to particular `/proc` entries or network information can vary depending on:

* kernel configuration
* filesystem mount options
* process visibility restrictions
* user permissions
* network namespace
* containerization
* security policies

A collection failure should therefore be treated as an observation about the environment rather than automatically interpreted as a host health failure.

This distinction will become important when the health-evaluation layer is implemented.

### Recommended First Run

For someone working with the repository for the first time:

```bash
git clone https://github.com/noklash/hostcheck.git
cd hostcheck

go test ./...
go vet ./...
go build ./...

go run ./experiments/cpu
go run ./experiments/filesystem
go run ./experiments/process
go run ./experiments/process-cpu
go run ./experiments/process-memory
go run ./experiments/network-addresses
go run ./experiments/network-routing
```

Then inspect the subsystem documentation:

```text
docs/memory-accounting.md
docs/filesystem-capacity.md
docs/process-memory-accounting.md
docs/network-addresses.md
docs/network-routing.md
```

The recommended approach is to understand the Linux interface first, run the relevant experiment, then inspect the collector implementation and tests.

That workflow is central to the project. hostcheck is not intended to hide Linux behavior behind a convenient command. It is intended to make that behavior understandable and then encode it carefully.


## Architecture

The intended architecture is:

```text
Linux kernel interfaces
        |
        v
    Collectors
        |
        v
  Host snapshot
        |
        v
 Health evaluation
        |
        v
      Output
```

Collectors are responsible for obtaining and parsing operating-system state.

The host snapshot layer will combine subsystem observations into a single host-level representation.

Health evaluation will interpret those observations using explicitly defined reliability semantics.

Output will present the resulting information without coupling collection to a particular presentation format.

This separation is deliberate. Raw kernel measurements should not be mixed with health policy or presentation logic.

## Linux Interfaces

hostcheck currently works directly with several Linux kernel interfaces:

```text
/proc/stat
    CPU accounting

/proc/meminfo
    System memory accounting

statfs
    Filesystem capacity and inode accounting

/proc/<pid>/
    Process statistics and memory accounting

rtnetlink
    Network addresses and routes

network interface metadata
    Interface identity and link information
```

The project deliberately keeps the Linux interface visible in the implementation.

Abstractions are introduced where they improve testability or separate collection from interpretation, not simply to hide the operating system.

## Repository Structure

```text
hostcheck/
├── internal/
│   ├── cpu/
│   │   ├── collector.go
│   │   ├── collector_test.go
│   │   ├── delta.go
│   │   ├── delta_test.go
│   │   ├── parser.go
│   │   ├── parser_test.go
│   │   ├── stat.go
│   │   ├── utilization.go
│   │   └── utilization_test.go
│   │
│   ├── filesystem/
│   │   ├── capacity.go
│   │   ├── capacity_test.go
│   │   ├── collector.go
│   │   ├── collector_test.go
│   │   └── stat.go
│   │
│   ├── memory/
│   │   ├── collector.go
│   │   ├── collector_test.go
│   │   ├── meminfo.go
│   │   ├── parser.go
│   │   └── parser_test.go
│   │
│   ├── network/
│   │   ├── address.go
│   │   ├── address_reader.go
│   │   ├── address_reader_test.go
│   │   ├── enumerate.go
│   │   ├── enumerate_test.go
│   │   ├── interface.go
│   │   ├── interface_reader.go
│   │   ├── interface_reader_test.go
│   │   ├── network.go
│   │   ├── network_test.go
│   │   ├── route.go
│   │   ├── route_reader.go
│   │   ├── route_reader_test.go
│   │   ├── statistics.go
│   │   ├── statistics_reader.go
│   │   └── statistics_test.go
│   │
│   └── process/
│       ├── collector.go
│       ├── collector_test.go
│       ├── delta.go
│       ├── delta_test.go
│       ├── enumerate.go
│       ├── enumerate_test.go
│       ├── memory.go
│       ├── memory_parser.go
│       ├── memory_parser_test.go
│       ├── memory_reader.go
│       ├── memory_reader_test.go
│       ├── parser.go
│       ├── parser_test.go
│       ├── process.go
│       ├── reader.go
│       ├── reader_test.go
│       ├── sample.go
│       ├── stat.go
│       ├── status.go
│       ├── utilization.go
│       └── utilization_test.go
│
├── docs/
│   ├── filesystem-capacity.md
│   ├── memory-accounting.md
│   ├── network-addresses.md
│   ├── network-routing.md
│   └── process-memory-accounting.md
│
├── experiments/
│   ├── cpu/
│   ├── filesystem/
│   ├── network-addresses/
│   ├── network-route-table/
│   ├── network-routing/
│   ├── network-rtnetlink/
│   ├── process/
│   ├── process-cpu/
│   └── process-memory/
│
├── go.mod
└── README.md
```

The repository is intentionally organized around Linux subsystems rather than around a large application framework.

## CPU Collection

The aggregate CPU record in `/proc/stat` has the form:

```text
cpu user nice system idle iowait irq softirq steal guest guest_nice
```

Linux reports these values as cumulative CPU time counters measured in clock ticks.

hostcheck currently calculates:

```text
total =
    user +
    nice +
    system +
    idle +
    iowait +
    irq +
    softirq +
    steal
```

and:

```text
busy =
    user +
    nice +
    system +
    irq +
    softirq +
    steal
```

Guest time is retained as raw information but is not independently added to the total because Linux CPU accounting already includes guest time in other counters.

CPU utilization is derived from the change in these counters between two observations.

The implementation also detects counter regression. A decrease in a cumulative counter is treated as invalid input rather than silently producing an incorrect utilization value.

## Memory Collection

The memory collector follows this flow:

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

The production collector opens `/proc/meminfo`.

The reader-based internal function provides a deterministic test boundary between filesystem access and parsing.

The parser ignores fields outside the current V1 model while validating all required fields.

Values are retained in the `kB` unit reported by Linux.

## Filesystem Collection

Filesystem collection uses the Linux `statfs` interface.

The collector reads filesystem-level capacity and inode information and stores the raw values in a `Stats` model.

Derived calculations are kept separate from collection.

For example:

```text
used blocks = total blocks - free blocks
```

and:

```text
available bytes = blocks available × block size
```

The implementation validates arithmetic assumptions and checks for overflow before returning derived values.

This distinction between collection and calculation makes the filesystem model easier to test and prevents invalid raw values from being silently converted into misleading results.

## Process Collection

Process collection works against the live `/proc` filesystem.

The general flow is:

```text
/proc
  |
  v
enumerate PIDs
  |
  v
read /proc/<pid>/stat
  |
  v
read /proc/<pid>/status
  |
  v
Process
```

The process collector intentionally handles the fact that `/proc` represents a changing system.

A process may exist during enumeration and disappear before its information can be read. hostcheck treats this as a normal lifecycle race and skips that process rather than failing the entire collection.

Process observations contain:

* process statistics
* whether the process is a kernel thread
* optional memory information

Kernel threads are represented explicitly rather than treated as malformed userspace processes.

Process CPU utilization is calculated from samples of cumulative CPU accounting, using the same general principle applied to system CPU accounting.

## Network Collection

Network collection builds a host network snapshot from several observations.

The flow is:

```text
network interfaces
       |
       v
interface identity
       |
       +----> interface addresses
       |
       v
network snapshot
       |
       +----> routing information
```

Interface enumeration identifies the available network interfaces.

Each interface can then be enriched with:

* hardware address
* operational state
* carrier state
* MTU
* optional speed
* optional duplex information
* addresses

Routing information is collected separately and retains the interface index used by the Linux kernel.

The network layer then preserves those kernel-level route attributes instead of immediately reducing them to a simplified "default gateway" representation.

This is important because Linux routing can contain multiple tables, route types, protocols, priorities, source prefixes, and multipath next hops.

### Multipath Routes

Routes may contain multiple next hops.

hostcheck represents these explicitly through:

```go
type NextHop struct {
    InterfaceIndex uint32
    Gateway        net.IP
    Hops           uint8
    Flags          uint8
}
```

This preserves the structure of multipath routing rather than flattening it into a single interface or gateway.

### Network Observation Time

The network snapshot records:

```go
ObservedAt time.Time
```

This identifies when network collection began.

The timestamp does not imply that every field in the snapshot was observed at exactly the same instant.

Network state is live and can change while the collector is running. Therefore the resulting snapshot is a best-effort observation rather than an atomic view of the entire networking subsystem.

## Experiments

Experiments are small programs used to investigate Linux behavior before that behavior is encoded into production collectors.

Current experiments include:

### CPU

```text
experiments/cpu/
```

Samples `/proc/stat`, calculates cumulative counter deltas, and derives aggregate CPU utilization.

### Filesystem

```text
experiments/filesystem/
```

Inspects filesystem capacity and inode information exposed through the Linux filesystem statistics interface.

### Process

```text
experiments/process/
```

Inspects raw process information from `/proc`.

### Process CPU

```text
experiments/process-cpu/
```

Investigates process CPU accounting and utilization across multiple samples.

### Process Memory

```text
experiments/process-memory/
```

Uses controlled memory mappings and workload phases to investigate how process memory measurements appear through Linux `/proc` interfaces.

### Network Addresses

```text
experiments/network-addresses/
```

Inspects interface addresses and the relationship between Linux network state and rtnetlink messages.

### Network Routing

```text
experiments/network-routing/
experiments/network-route-table/
```

Investigates Linux routing information, route tables, route attributes, and IPv6 routing behavior.

### Network rtnetlink

```text
experiments/network-rtnetlink/
```

Inspects raw rtnetlink information before it is represented by the production network collector.

The experiments are part of the engineering process rather than disposable examples. Their purpose is to establish what Linux actually reports before the project assigns meaning to those observations.

## Documentation

Subsystem-specific documentation is kept separately from the implementation.

Current documentation includes:

```text
docs/memory-accounting.md
docs/filesystem-capacity.md
docs/process-memory-accounting.md
docs/network-addresses.md
docs/network-routing.md
```

The documentation explains Linux behavior, accounting semantics, implementation decisions, and important limitations discovered during development.

The README provides the project-level view. Detailed Linux behavior belongs in the subsystem documentation.

## Testing

Tests are written alongside each subsystem.

The project currently tests:

* valid parsing
* malformed input
* invalid numeric values
* invalid units
* missing fields
* duplicate fields
* CPU counter regressions
* CPU utilization calculations
* filesystem arithmetic
* filesystem overflow conditions
* filesystem collector behavior
* process enumeration
* process lifecycle races
* process CPU calculations
* process memory parsing
* kernel-thread memory behavior
* network interface collection
* network address parsing
* route parsing
* multipath route handling
* collector failures
* reader failures

Run the complete test suite with:

```bash
go test ./...
```

Run static analysis with:

```bash
go vet ./...
```

Format the source tree with:

```bash
gofmt -w .
```

Check for whitespace errors with:

```bash
git diff --check
```

## Engineering Approach

Each subsystem follows the same general workflow:

```text
Research
   |
   v
Experiment
   |
   v
Implement
   |
   v
Test
   |
   v
Document
   |
   v
Validate
   |
   v
Commit
```

The goal is to avoid treating Linux interfaces as opaque data sources.

For each subsystem, the project asks:

1. What does Linux actually expose?
2. What do the fields mean?
3. Which values are cumulative counters?
4. Which values are derived?
5. What can change during collection?
6. What failure modes are normal?
7. What assumptions are safe to encode?
8. What should remain raw information?
9. What belongs to health policy rather than collection?

This approach keeps the implementation grounded in actual operating-system behavior.

## Current Scope

The current scope is intentionally limited to host-level observation.

Included:

* Linux CPU accounting
* system memory accounting
* filesystem capacity and inode accounting
* process enumeration and accounting
* network interface state
* network addresses
* network routing
* controlled Linux experiments
* deterministic tests
* subsystem documentation

Not yet included:

* daemon mode
* background scheduling
* Prometheus exporters
* Grafana dashboards
* Kubernetes integration
* Docker integration
* cloud integrations
* remote collection
* alerting
* distributed collection
* persistent storage
* automatic remediation

These may become relevant later, but they are not prerequisites for understanding the host itself.

## Next Engineering Phase

The next phase is to review the existing subsystem models as one host observation system and define the host snapshot boundary.

This includes deciding:

* what constitutes one host observation
* which subsystem observations belong together
* how collection timestamps are represented
* how partial collection failures are represented
* how live-system races are represented
* which values are raw measurements
* which values are derived
* which observations can legitimately contribute to a future health decision

Health evaluation will follow that model review.

The project will not introduce arbitrary CPU, memory, disk, or network thresholds simply because those metrics are available.

Health semantics should be based on documented operating-system behavior and explicit engineering decisions.

## Design Constraints

The project deliberately favors:

* Linux-native interfaces
* small packages
* explicit data models
* deterministic tests
* controlled experiments
* clear failure handling
* documented assumptions
* raw observations before interpretation
* small logical commits
* readable implementation over premature abstraction

The project deliberately avoids:

* framework-heavy architecture
* premature distributed-system design
* infrastructure integrations before the host model is understood
* arbitrary health thresholds
* hiding Linux behavior behind unnecessary abstractions

## Development Workflow

Before committing changes:

```bash
gofmt -w .
go test ./...
go vet ./...
git diff --check
git status --short --branch
```

Commits should represent one logical engineering change.

The repository history is intended to show how the system was investigated and built, not just the final state.

## Project Direction

hostcheck is being developed as a systems engineering project rather than as a generic monitoring application.

The long-term direction is to move upward through the stack:

```text
Linux interfaces
      |
      v
Host observation
      |
      v
Host model
      |
      v
Reliability semantics
      |
      v
Health evaluation
      |
      v
Structured output
      |
      v
Operational integration
```

Each layer should be understandable before the next layer is added.

> First understand the system. Then master the tools. Finally, build the platform.