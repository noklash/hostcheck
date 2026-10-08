# hostcheck

A Linux host health and reliability agent written in Go.

hostcheck reads operating-system interfaces exposed by Linux and turns them into structured host observations, derived measurements, health assessments, and explainable host-level results.

The project is built from the kernel interfaces upward. Each subsystem is researched, experimentally verified, implemented, tested, documented, and then integrated into the larger host model.

The guiding principle is:

> Understand the system first. Then build the tool.

hostcheck currently operates as a **one-shot Linux host health agent**. It collects a host snapshot, evaluates explicitly configured health policies, produces explainable results, and exposes those results through human-readable or structured JSON output.

Continuous monitoring, exporters, dashboards, orchestration integrations, remote collection, and automatic remediation remain outside the current V1 scope.

---

## Objective

Build a small but technically defensible Linux host inspection and health agent that can answer reliability questions directly from the operating system.

The project has two related objectives:

1. Understand the Linux interfaces that expose host state.
2. Encode that understanding into a reliable, testable, and explainable system.

hostcheck deliberately separates:

```text
Linux observation
      |
      v
Derived measurements
      |
      v
Health policy
      |
      v
Assessment
      |
      v
Aggregation
      |
      v
Application output
```

This separation prevents raw operating-system measurements from becoming accidentally coupled to arbitrary health thresholds or presentation logic.

---

## Current State

hostcheck currently has the following major layers:

```text
Linux kernel / operating-system interfaces
                  |
                  v
             Collectors
                  |
                  v
          Host Snapshot
                  |
                  v
       Derived observations
                  |
                  v
        Health evaluation
                  |
                  v
       Host-level Result
                  |
          +-------+-------+
          |               |
          v               v
     Human output      JSON output
```

The project currently collects five major Linux subsystems:

* CPU accounting
* Memory
* Filesystem capacity and inode usage
* Processes
* Network interfaces, addresses, and routes

The collectors remain responsible for observing Linux state.

Derived calculations remain separate from raw collection.

Health evaluation applies explicit policies to observations.

The aggregation layer combines individual assessments into a host-level result containing:

* status
* coverage
* individual assessments

The application layer in `cmd/hostcheck` connects these components into a usable one-shot command.

---

# V1 Application

The current application entrypoint is:

```text
cmd/hostcheck/
```

Run the host health check with:

```bash
go run ./cmd/hostcheck
```

A successful run currently produces output similar to:

```text
hostcheck
observed_at: 2026-10-08T07:58:07+01:00
status: ok
coverage: complete

[ok] memory
  available memory capacity is within the configured policy
  available_percent=48.21

[ok] filesystem
  available filesystem capacity is within the configured policy
  available_percent=49.78

[ok] filesystem_inodes
  available filesystem inodes are within the configured policy
  available_inode_percent=88.60
```

The application currently uses explicit default policies for:

* available memory
* filesystem available capacity
* filesystem available inodes

The default thresholds are:

```text
degraded: below 20%
critical: below 10%
```

These are application defaults, not universal Linux health truths.

Process-state and network-interface or route policies are not enabled by default because those checks require operator-specific expectations about which processes, interfaces, and routes should exist on a particular host.

---

## JSON Output

The application supports structured JSON output:

```bash
go run ./cmd/hostcheck --json
```

Example:

```json
{
  "observed_at": "2026-10-08T07:59:12+01:00",
  "status": "ok",
  "coverage": "complete",
  "assessments": [
    {
      "subject": "memory",
      "availability": "assessable",
      "status": "ok",
      "reason": "available memory capacity is within the configured policy",
      "evidence": [
        "available_percent=50.29"
      ]
    },
    {
      "subject": "filesystem",
      "availability": "assessable",
      "status": "ok",
      "reason": "available filesystem capacity is within the configured policy",
      "evidence": [
        "available_percent=49.79"
      ]
    },
    {
      "subject": "filesystem_inodes",
      "availability": "assessable",
      "status": "ok",
      "reason": "available filesystem inodes are within the configured policy",
      "evidence": [
        "available_inode_percent=88.60"
      ]
    }
  ]
}
```

JSON is intended to provide a stable integration boundary without forcing the internal health model to depend on a particular output format.

The JSON representation preserves:

* observation timestamp
* host status
* coverage
* assessment subject
* assessment availability
* assessment status when applicable
* reason
* evidence

An unavailable assessment does not automatically become a failed health assessment. Its availability is represented explicitly.

---

## Exit Codes

The command provides process exit codes suitable for shell automation and future operational integration.

Current semantics are:

| Result                     | Exit code |
| -------------------------- | --------: |
| OK                         |       `0` |
| Degraded                   |       `1` |
| Critical                   |       `2` |
| Unavailable                |       `1` |
| Invalid application result |       `1` |

The exit code communicates the host-level result without replacing the detailed assessment information.

JSON or human-readable output should be used when the caller needs the actual explanation.

---

# Architecture

The current architecture is:

```text
                    Linux
                     |
        +------------+------------+
        |            |            |
       /proc       statfs      rtnetlink
        |            |            |
        v            v            v
     Collectors   Collectors   Collectors
        |            |            |
        +------------+------------+
                     |
                     v
              Host Snapshot
                     |
                     v
          Derived Observations
                     |
                     v
             Health Policies
                     |
                     v
              Assessments
                     |
                     v
                Aggregate
                     |
                     v
             Health Result
                /       \
               /         \
              v           v
          Human          JSON
           CLI           Output
```

The important architectural boundaries are:

### Collection

Collect operating-system state without deciding whether that state is healthy.

### Derivation

Calculate values that require interpretation of raw counters or filesystem values.

Examples include:

* CPU utilization
* available memory percentage
* filesystem available capacity percentage
* available inode percentage
* process CPU utilization

### Policy

Define explicit rules for determining whether an observation is acceptable.

Policies are caller-supplied rather than hidden inside collectors.

### Assessment

Turn an observation and a policy into an explainable assessment.

An assessment contains:

```go
type Assessment struct {
    Subject      string
    Availability Availability
    Status       Status
    Reason       string
    Evidence     []string
}
```

### Aggregation

Combine individual assessments into a host-level result.

The result contains:

```go
type Result struct {
    Status      Status
    Coverage    Coverage
    Assessments []Assessment
}
```

### Application

The command-line application connects collection, evaluation, aggregation, and output.

The application should remain thin.

It should not become the place where Linux semantics, collection logic, or health policy are hidden.

---

# Host Snapshot

The host boundary is represented by:

```go
type Snapshot struct {
    ObservedAt time.Time

    CPU        *cpu.Stat
    Memory     *memory.MemInfo
    Filesystem *filesystem.Stats
    Processes  []process.Process
    Network    *network.Network

    Errors []CollectionError
}
```

The snapshot provides a common integration boundary for subsystem observations.

It also records collection failures explicitly:

```go
type CollectionError struct {
    Subsystem string
    Err       error
}
```

Collection is currently sequential.

`ObservedAt` therefore represents the snapshot boundary rather than an assertion that every subsystem was observed at exactly the same instant.

A Linux host is changing while it is being inspected. The snapshot is consequently a best-effort observation of the host at a point in the collection process.

---

# Health Evaluation

Health evaluation is deliberately separate from collection.

The health layer currently supports:

* memory available capacity
* filesystem available capacity
* filesystem available inode capacity
* process-state policy
* network-interface policy
* network-route policy
* host-level aggregation

CPU utilization assessment exists as a standalone health capability because meaningful CPU utilization requires multiple observations.

It is intentionally not treated as a single-snapshot health measurement.

---

## Availability

An observation can be either:

```text
assessable
```

or:

```text
unassessable
```

An unavailable observation does not automatically become `critical`.

For example, if a subsystem could not be collected because of an environment-specific failure, the health system preserves that distinction.

This prevents:

```text
collection failed
        =
host is unhealthy
```

from becoming an automatic assumption.

---

## Health Status

Assessable observations can have one of three statuses:

```text
ok
degraded
critical
```

The aggregation order is:

```text
critical
   ^
degraded
   ^
ok
```

Critical assessments dominate degraded assessments.

Degraded assessments dominate OK assessments.

Unassessable assessments do not automatically become critical.

---

## Coverage

The host-level result also reports coverage.

Coverage can be:

```text
complete
partial
unavailable
```

### Complete

All supplied assessments were assessable.

### Partial

At least one assessment was assessable and at least one was unavailable.

### Unavailable

No supplied assessment was assessable.

This distinction is important because:

```text
critical + unavailable
```

is different from:

```text
critical + complete
```

A host can have a real critical condition while other parts of the host could not be evaluated.

---

# Memory Health

Memory health uses Linux's `MemAvailable` value rather than `MemFree`.

The derived observation is:

```text
available_percent =
    MemAvailable / MemTotal * 100
```

The current default application policy is:

```text
degraded: below 20%
critical: below 10%
```

The policy evaluates available memory capacity.

It does not claim to detect:

* historical memory pressure
* reclaim trends
* future out-of-memory conditions
* swap trends
* memory pressure duration

Those require additional observations or time-series information.

---

# Filesystem Health

Filesystem health currently evaluates:

* available filesystem capacity
* available filesystem inodes

The policies use explicit percentage thresholds.

The application default is:

```text
degraded: below 20%
critical: below 10%
```

Capacity and inode exhaustion are evaluated separately because a filesystem can have:

```text
plenty of free bytes
```

while simultaneously having:

```text
very few free inodes
```

These are different resource constraints.

---

# Process Health

The process subsystem supports process-state health assessment.

The current model can evaluate observed process states such as uninterruptible sleep.

The assessment is intentionally limited to what one collection can establish.

For example, observing a process in `D` state means the process was observed in that state during collection.

It does not prove:

* how long the process has been there
* whether the condition is persistent
* whether the process is permanently blocked

Those claims require repeated observations.

Process-state policies are therefore optional and are not enabled by the default CLI policy.

---

# Network Health

The network health layer currently supports explicit policies for:

* expected interfaces
* interface operational state
* optional carrier requirements
* expected routes
* route interface selection
* route attributes
* multipath route expectations

Network health is deliberately about **local network configuration and observed interface state**.

It does not currently claim to establish:

* gateway reachability
* DNS availability
* Internet access
* remote service reachability
* application-layer connectivity

Those would require an explicit active probe model.

No active network reachability probes are currently part of V1.

---

# CPU

CPU accounting is collected from:

```text
/proc/stat
```

The aggregate CPU record has the form:

```text
cpu user nice system idle iowait irq softirq steal guest guest_nice
```

Linux reports these values as cumulative CPU time counters measured in clock ticks.

hostcheck preserves the raw counters and derives utilization from changes between observations.

The current model distinguishes:

* user time
* nice time
* system time
* idle time
* I/O wait
* hardware interrupt time
* software interrupt time
* steal time
* guest time
* guest-nice time

Guest time is retained as raw information but is not independently added to the total because Linux already accounts for guest time within other CPU counters.

CPU utilization is calculated from counter deltas:

```text
delta_total = total_after - total_before

delta_busy = busy_after - busy_before

utilization = delta_busy / delta_total
```

The implementation detects counter regression.

A cumulative CPU counter moving backwards is treated as invalid input rather than silently producing an incorrect utilization value.

CPU health remains separate from single-snapshot evaluation because utilization is inherently a multi-sample observation.

---

# Memory Collection

Memory statistics are collected from:

```text
/proc/meminfo
```

The current model includes:

* `MemTotal`
* `MemAvailable`
* `MemFree`
* `Buffers`
* `Cached`
* `SwapTotal`
* `SwapFree`

The parser validates:

* required fields
* numeric values
* units
* duplicate fields
* missing fields

The production collector reads:

```text
/proc/meminfo
```

The reader-based implementation provides a deterministic boundary between filesystem access and parsing.

Values are retained using the unit reported by Linux.

Detailed memory semantics are documented in:

```text
docs/memory-accounting.md
```

A controlled 512 MiB memory-pressure experiment was also performed to understand how available memory, free memory, cache, and swap changed under workload.

---

# Filesystem Collection

Filesystem statistics are collected using Linux `statfs`.

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

The implementation deliberately preserves the distinction between:

```text
free blocks
```

and:

```text
blocks available to an unprivileged process
```

These represent different Linux filesystem semantics.

Derived arithmetic is kept separate from raw collection.

The implementation checks arithmetic assumptions and overflow conditions before returning derived values.

Detailed filesystem semantics are documented in:

```text
docs/filesystem-capacity.md
```

---

# Process Collection

Process observations are collected from the live `/proc` filesystem.

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

The process subsystem currently supports:

* process enumeration
* `/proc/<pid>/stat` parsing
* process lifecycle race handling
* process CPU accounting
* process CPU utilization
* process memory accounting
* kernel-thread identification
* process-state information

A process can disappear between enumeration and individual `/proc` reads.

This is normal behavior for a live Linux system.

hostcheck treats that lifecycle race as an expected property of `/proc` and skips processes that disappear during collection rather than failing the entire collection.

Kernel threads are included in process collection.

Their `Kthread` field is set to `true`.

Userspace memory information is unavailable for kernel threads because they do not expose the same userspace memory accounting through `/proc/<pid>/status`.

Detailed process memory semantics are documented in:

```text
docs/process-memory-accounting.md
```

Controlled experiments are used to investigate process CPU and memory behavior under known workloads.

---

# Network Collection

Network state is collected from Linux network interfaces and kernel routing information.

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
* routes
* multipath route next hops

The network snapshot also records:

```go
ObservedAt time.Time
```

This identifies the observation boundary for the network collection.

It does not imply that every network field was observed at exactly the same instant.

Network state can change while collection is taking place.

---

## Network Addresses

Interface addresses are collected through Linux networking interfaces and rtnetlink.

An address retains:

```go
type Address struct {
    IP        net.IP
    PrefixLen int
}
```

The model intentionally avoids inferring Linux address scope from Go's generic IP classification.

Linux networking semantics should come from Linux networking data.

The implementation also avoids parsing human-readable `ip` command output as its primary collection mechanism.

Detailed address behavior is documented in:

```text
docs/network-addresses.md
```

---

## Network Routes

Routes are collected through rtnetlink.

The route model retains kernel-level attributes including:

* address family
* destination
* destination prefix length
* source
* source prefix length
* gateway
* output interface index
* priority
* routing table
* protocol
* scope
* route type
* route flags
* multipath next hops

The project intentionally avoids reducing Linux routing to a simplistic:

```text
default gateway
```

model.

Linux routing can contain multiple routing tables, protocols, priorities, source prefixes, route types, and multipath next hops.

Detailed routing semantics are documented in:

```text
docs/network-routing.md
```

---

## Multipath Routes

Linux routes can contain multiple next hops.

hostcheck preserves those next hops explicitly.

The model includes information such as:

```go
type NextHop struct {
    InterfaceIndex uint32
    Gateway        net.IP
    Hops           uint8
    Flags          uint8
}
```

This preserves the structure of multipath routing instead of flattening the route into a single gateway or interface.

---

# Linux Interfaces

hostcheck currently works directly with several Linux operating-system interfaces:

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

The project deliberately keeps these interfaces visible in the implementation.

Abstractions are introduced where they improve:

* testability
* separation of concerns
* portability within the Linux model
* integration boundaries

They are not introduced merely to hide Linux behavior.

---

# Experiments

Experiments are small programs used to investigate Linux behavior before that behavior is encoded into production collectors.

Current experiments include:

```text
experiments/cpu/
experiments/filesystem/
experiments/network-addresses/
experiments/network-route-table/
experiments/network-routing/
experiments/network-rtnetlink/
experiments/process/
experiments/process-cpu/
experiments/process-memory/
```

Run an experiment with:

```bash
go run ./experiments/<experiment>
```

For example:

```bash
go run ./experiments/cpu
```

---

## CPU Experiment

```text
experiments/cpu/
```

Investigates:

* `/proc/stat`
* cumulative CPU counters
* counter deltas
* aggregate utilization
* sampling behavior

---

## Filesystem Experiment

```text
experiments/filesystem/
```

Investigates:

* filesystem capacity
* block accounting
* inode accounting
* Linux filesystem statistics

---

## Process Experiment

```text
experiments/process/
```

Investigates:

* process enumeration
* `/proc/<pid>/stat`
* `/proc/<pid>/status`
* process lifecycle behavior

---

## Process CPU Experiment

```text
experiments/process-cpu/
```

Investigates process CPU accounting and utilization across multiple samples.

---

## Process Memory Experiment

```text
experiments/process-memory/
```

Uses controlled memory mappings and workload phases to investigate how process memory measurements appear through Linux `/proc` interfaces.

---

## Network Address Experiment

```text
experiments/network-addresses/
```

Investigates interface addresses and their representation through Linux networking interfaces and rtnetlink.

---

## Network Routing Experiments

```text
experiments/network-routing/
experiments/network-route-table/
```

Investigate:

* routing information
* route tables
* route attributes
* IPv4 routing
* IPv6 routing

---

## Network rtnetlink Experiment

```text
experiments/network-rtnetlink/
```

Inspects rtnetlink information before it is represented by the production network collector.

The experiments are part of the engineering process.

They are used to establish what Linux actually reports before the project assigns meaning to those observations.

---

# Testing

Testing is performed alongside implementation.

The project currently tests areas including:

* valid parsing
* malformed input
* invalid numeric values
* invalid units
* missing fields
* duplicate fields
* CPU counter regression
* CPU utilization
* filesystem arithmetic
* filesystem overflow
* filesystem collector behavior
* process enumeration
* process lifecycle races
* process CPU calculations
* process memory parsing
* kernel-thread behavior
* network interface collection
* network address parsing
* route parsing
* multipath route handling
* collector failures
* reader failures
* health assessment validation
* health aggregation
* memory health policies
* filesystem health policies
* inode health policies
* process-state health
* network-interface health
* network-route health
* host snapshot evaluation
* application policy defaults
* application exit codes
* JSON serialization
* unavailable JSON results

Run the complete test suite:

```bash
go test ./...
```

Run static analysis:

```bash
go vet ./...
```

Format Go source:

```bash
gofmt -w .
```

Check for whitespace errors:

```bash
git diff --check
```

---

# Development Validation

The normal validation sequence is:

```bash
gofmt -w .
go test ./...
go vet ./...
git diff --check
git status --short --branch
```

For application changes, also run:

```bash
go run ./cmd/hostcheck
go run ./cmd/hostcheck --json
```

The human-readable and JSON modes should represent the same underlying health result.

---

# Running the Project

## Requirements

hostcheck currently requires:

* Linux
* Go
* access to the Linux interfaces used by the collectors

The project is developed and tested directly on Linux because several collectors depend on Linux-specific interfaces such as:

```text
/proc
statfs
rtnetlink
```

Check the installed Go version:

```bash
go version
```

---

## Clone the Repository

```bash
git clone https://github.com/noklash/hostcheck.git
cd hostcheck
```

---

## Run the Complete Test Suite

```bash
go test ./...
```

---

## Run Static Analysis

```bash
go vet ./...
```

---

## Build the Repository

```bash
go build ./...
```

---

## Run the Host Health Agent

Human-readable output:

```bash
go run ./cmd/hostcheck
```

Structured JSON:

```bash
go run ./cmd/hostcheck --json
```

---

## Run Experiments

Examples:

```bash
go run ./experiments/cpu
go run ./experiments/filesystem
go run ./experiments/process
go run ./experiments/process-cpu
go run ./experiments/process-memory
go run ./experiments/network-addresses
go run ./experiments/network-routing
```

---

# Inspecting Linux Interfaces Directly

The collectors are intentionally based on operating-system interfaces that can also be inspected from the shell.

CPU:

```bash
cat /proc/stat
```

Memory:

```bash
cat /proc/meminfo
```

Process information:

```bash
cat /proc/self/stat
cat /proc/self/status
```

Filesystem:

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

IPv4 routes:

```bash
ip route
```

IPv6 routes:

```bash
ip -6 route
```

These commands are useful when investigating collector behavior because they provide a human-readable view of operating-system state.

They are investigation tools.

They are not the primary data source for the production collectors.

---

# Using the Internal Packages

The implementation is organized around Linux subsystems:

```text
internal/cpu
internal/memory
internal/filesystem
internal/process
internal/network
internal/health
internal/host
```

The architecture intentionally keeps collection separate from interpretation.

For example:

```text
/proc/stat
    |
    v
CPU collector
    |
    v
cpu.Stat
    |
    v
CPU derivation
    |
    v
CPU health policy
```

Memory follows the same general separation:

```text
/proc/meminfo
    |
    v
Memory collector
    |
    v
memory.MemInfo
    |
    v
Derived memory observation
    |
    v
Memory health policy
```

The same principle applies to filesystem, process, and network information.

---

# Repository Structure

The repository is organized around Linux subsystems and engineering boundaries rather than around a large application framework.

A simplified structure is:

```text
hostcheck/
|
├── cmd/
│   └── hostcheck/
│       ├── main.go
│       ├── main_test.go
│       ├── json.go
│       └── json_test.go
|
├── internal/
│   ├── cpu/
│   ├── filesystem/
│   ├── health/
│   ├── host/
│   ├── memory/
│   ├── network/
│   └── process/
|
├── docs/
│   ├── filesystem-capacity.md
│   ├── health-evaluation.md
│   ├── memory-accounting.md
│   ├── network-addresses.md
│   ├── network-routing.md
│   └── process-memory-accounting.md
|
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
|
├── go.mod
└── README.md
```

The exact file layout will evolve as the project develops.

The important structural boundaries are:

```text
cmd
    application boundary

internal/* collectors
    Linux observation

internal/host
    host-level integration

internal/health
    interpretation and reliability semantics

experiments
    Linux investigation

docs
    engineering knowledge and decisions
```

---

# Health Evaluation Model

The health layer intentionally separates observation from policy.

A memory collector might report:

```text
MemAvailable = 4.8 GiB
MemTotal     = 8.0 GiB
```

The collector should not decide:

```text
4.8 GiB is healthy
```

Instead, the health layer receives an explicit policy and evaluates the derived observation against it.

This makes the same observation reusable under different operational policies.

---

## Assessment

An assessment represents one health decision:

```go
type Assessment struct {
    Subject      string
    Availability Availability
    Status       Status
    Reason       string
    Evidence     []string
}
```

The result is intentionally explainable.

A status alone is insufficient.

For example:

```text
status: degraded
```

is less useful than:

```text
status: degraded

reason:
available memory capacity is below the configured degraded threshold

evidence:
available_percent=15.25
```

The application therefore preserves both the decision and the evidence behind it.

---

# Snapshot Semantics

A host snapshot is not treated as a perfect atomic representation of the machine.

The host is changing while it is being observed.

For example:

```text
process enumeration
       |
       | process exits
       v
process read
```

or:

```text
network interface collection
       |
       | route changes
       v
route collection
```

The implementation therefore treats collection as a best-effort observation.

The snapshot timestamp represents the observation boundary.

This is an important distinction from claiming that all subsystem measurements were captured at one exact instant.

---

# Failure Handling

Failure behavior is designed around the difference between:

```text
the host is unhealthy
```

and:

```text
the host could not be fully observed
```

A collection failure is therefore recorded as a collection error where appropriate.

Health evaluation can then represent the corresponding assessment as:

```text
unassessable
```

rather than automatically converting the failure into:

```text
critical
```

This distinction becomes especially important on systems where:

* `/proc` visibility is restricted
* processes disappear during enumeration
* network namespaces differ
* security policies restrict access
* filesystem information behaves differently
* specific interfaces are unavailable

---

# Permissions and Environment

Most collectors can operate using information available to ordinary Linux users.

However, behavior can vary depending on:

* kernel configuration
* `/proc` mount options
* process visibility restrictions
* user permissions
* network namespace
* containerization
* security policies
* filesystem configuration

The collector should not assume that every Linux host exposes exactly the same information.

Where information is unavailable, the implementation should preserve that fact rather than inventing a value.

---

# Engineering Approach

Each subsystem follows the general workflow:

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
Integrate
   |
   v
Commit
```

The project deliberately avoids starting with a large monitoring framework.

For each subsystem, the engineering questions are:

1. What does Linux actually expose?
2. What does each field mean?
3. Which values are cumulative counters?
4. Which values are derived?
5. What can change during collection?
6. What failure modes are normal?
7. Which assumptions are safe to encode?
8. Which information should remain raw?
9. What belongs to collection?
10. What belongs to derivation?
11. What belongs to health policy?
12. What claims can the available observations actually support?

This keeps the implementation grounded in operating-system behavior.

---

# Design Principles

hostcheck favors:

* Linux-native interfaces
* small packages
* explicit data models
* deterministic tests
* controlled experiments
* clear failure handling
* documented assumptions
* raw observations before interpretation
* explicit health policies
* explainable assessments
* small logical commits
* readable implementation
* deliberate abstraction

hostcheck deliberately avoids:

* framework-heavy architecture
* premature distributed-system design
* arbitrary health thresholds
* unnecessary abstraction
* parsing human-readable shell output as the primary data source
* infrastructure integrations before the host model is understood
* active network probes without a defined probe model
* treating unavailable observations as automatic failures

---

# Documentation

Subsystem-specific documentation is kept separately from the implementation.

Current documentation includes:

```text
docs/memory-accounting.md
docs/filesystem-capacity.md
docs/process-memory-accounting.md
docs/network-addresses.md
docs/network-routing.md
docs/health-evaluation.md
```

The documentation captures:

* Linux behavior
* accounting semantics
* implementation decisions
* assumptions
* experiments
* limitations
* health semantics
* architectural boundaries

The README provides the project-level view.

Detailed Linux behavior belongs in the subsystem documentation.

---

# Experiments as Engineering Evidence

Experiments are not disposable examples.

They are part of the project's engineering process.

The purpose of an experiment is to answer a question about Linux before that behavior becomes an assumption in production code.

The general process is:

```text
Question
   |
   v
Linux behavior
   |
   v
Controlled experiment
   |
   v
Observation
   |
   v
Model
   |
   v
Implementation
   |
   v
Test
```

This is particularly important for areas such as:

* CPU accounting
* memory accounting
* process memory
* process lifecycle
* filesystem capacity
* network routing
* rtnetlink behavior

The project prefers evidence from the operating system over assumptions inherited from generic monitoring tools.

---

# Current Scope

The current V1 scope includes:

* Linux CPU accounting
* CPU counter derivation
* system memory accounting
* derived memory capacity
* filesystem capacity
* filesystem inode accounting
* process enumeration
* process CPU accounting
* process CPU utilization
* process memory accounting
* process-state observations
* network interface state
* network addresses
* network routing
* multipath route representation
* host snapshots
* collection error representation
* health policies
* health assessments
* host-level health aggregation
* one-shot CLI execution
* human-readable output
* structured JSON output
* process exit codes
* deterministic tests
* controlled Linux experiments
* subsystem documentation

---

# Deliberately Out of Scope for V1

The following are not currently part of the V1 application:

* daemon mode
* continuous background monitoring
* Prometheus exporters
* Grafana dashboards
* Kubernetes integration
* Docker integration
* cloud-provider integrations
* remote collection
* distributed collection
* persistent metric storage
* alerting infrastructure
* automatic remediation
* active network reachability probes
* DNS health probes
* Internet reachability checks
* application-level service probes

These may become relevant later.

They are not prerequisites for building a technically defensible host health model.

---

# CPU Sampling Boundary

CPU utilization is fundamentally different from measurements such as:

```text
MemTotal
MemAvailable
filesystem capacity
inode availability
```

CPU utilization requires at least two observations.

Therefore:

```text
CPU counter
    |
    v
sample 1
    |
    | time
    v
sample 2
    |
    v
delta
    |
    v
utilization
```

The project deliberately keeps this multi-sample model separate from the single-snapshot health path.

A future continuous or sampled evaluation layer can build on this without forcing the current snapshot model to pretend that a single CPU counter represents utilization.

---

# Network Reachability Boundary

The current network health model evaluates local network state.

For example:

```text
expected interface exists
expected interface is operational
expected route exists
expected route uses expected interface
```

These observations do not prove:

```text
gateway reachable
DNS working
Internet reachable
remote API reachable
application healthy
```

Those are different questions.

A future active probe subsystem would need explicit semantics around:

* destination
* protocol
* timeout
* retry behavior
* network namespace
* failure classification
* probe scheduling
* permission requirements

Until that model exists, hostcheck does not pretend that local routing information is equivalent to connectivity.

---

# Configuration

The application currently uses explicit built-in default policies for the basic V1 health checks.

It does not yet introduce a large configuration system.

This is intentional.

Configuration becomes useful when there is a clear operational need to customize:

* thresholds
* expected interfaces
* expected routes
* process-state policies
* output behavior

The project should not introduce configuration architecture merely because production monitoring tools commonly have one.

The current application boundary is deliberately small.

---

# Recommended First Run

For someone working with the repository for the first time:

```bash
git clone https://github.com/noklash/hostcheck.git
cd hostcheck

go test ./...
go vet ./...
go build ./...

go run ./cmd/hostcheck
go run ./cmd/hostcheck --json
```

Then explore the experiments:

```bash
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
docs/health-evaluation.md
```

The recommended workflow is:

```text
Understand the Linux interface
          |
          v
Run the experiment
          |
          v
Inspect the collector
          |
          v
Read the tests
          |
          v
Read the documentation
          |
          v
Understand the health policy
          |
          v
Run the application
```

---

# Development Workflow

Before committing changes:

```bash
gofmt -w .
go test ./...
go vet ./...
git diff --check
git status --short --branch
```

For changes involving the application boundary:

```bash
go run ./cmd/hostcheck
go run ./cmd/hostcheck --json
```

Commits should represent one logical engineering change.

Examples:

```text
network: collect addresses and routes via rtnetlink
memory: derive available capacity
health: aggregate host assessments
cmd: add host health check entrypoint
cmd: add JSON output
docs: reconcile README with V1
```

The repository history is intended to show how the system was investigated and built, not merely what the final code looks like.

---

# Project Direction

hostcheck is being developed as a systems engineering project rather than as a generic monitoring application.

The direction is to move upward through the stack:

```text
Linux interfaces
       |
       v
Host observation
       |
       v
Derived measurements
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
Structured result
       |
       v
Application output
       |
       v
Operational integration
```

Each layer should be understandable before the next layer is added.

The current V1 application boundary is intentionally small:

```text
collect
   |
   v
snapshot
   |
   v
evaluate
   |
   v
aggregate
   |
   v
report
```

The next work should strengthen the existing boundaries rather than immediately expanding the system into a daemon, exporter, distributed agent, or orchestration platform.

---

# V1 Completion Criteria

The first usable V1 should be judged by engineering behavior rather than by the number of features.

The core criteria are:

* Linux observations are collected correctly.
* Derived values have explicit semantics.
* Collection failures are represented honestly.
* Health policies are explicit.
* Assessments are explainable.
* Aggregation preserves coverage information.
* The application can perform a complete one-shot evaluation.
* Human-readable output is useful to an operator.
* JSON output is usable by another program.
* Exit codes communicate the host-level result.
* Tests cover important failure modes.
* Documentation reflects the actual implementation.
* The repository remains understandable without requiring a larger framework.

The goal is not to produce the largest monitoring agent.

The goal is to produce a small host health system whose behavior can be explained from the Linux interfaces upward.

---

# Final Architecture

The current project can be understood as:

```text
                         Linux
                          |
          +---------------+---------------+
          |               |               |
        /proc           statfs        rtnetlink
          |               |               |
          v               v               v
       CPU /          Filesystem       Network
      Memory /                         Interface /
      Process                           Address /
                                        Route
          |               |               |
          +---------------+---------------+
                          |
                          v
                   Host Snapshot
                          |
                          v
                Derived Observations
                          |
                          v
                  Health Policies
                          |
                          v
                    Assessments
                          |
                          v
                     Aggregate
                          |
                          v
                    Health Result
                     /         \
                    /           \
                   v             v
              Human CLI        JSON
                   \             /
                    \           /
                     v         v
                    Operator /
                    Integration
```

The architecture is intentionally boring where it should be boring.

Linux observation should be explicit.

Derived measurements should be testable.

Health policy should be visible.

Assessments should explain themselves.

Aggregation should preserve uncertainty.

The CLI should remain thin.

Output should not determine the internal model.

---

# Philosophy

hostcheck is not intended to hide Linux behind a convenient command.

It is intended to make Linux behavior understandable first, then encode that understanding carefully.

The project therefore moves from:

```text
kernel interface
```

to:

```text
observation
```

to:

```text
measurement
```

to:

```text
policy
```

to:

```text
health decision
```

to:

```text
operational result
```

without skipping the layers in between.

> First understand the system. Then master the tools. Finally, build the platform.
