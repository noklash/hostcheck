# hostcheck

**A Linux host health and reliability agent written in Go.**

hostcheck collects operating-system observations directly from Linux interfaces and turns them into structured host snapshots, derived measurements, health assessments, and explainable host-level results.

The project is built from the kernel interfaces upward. Each subsystem is researched, experimentally verified, implemented, tested, documented, and integrated into the larger host model.

The guiding principle is:

> First understand the system. Then build the tool.

hostcheck currently operates as a **one-shot Linux host health agent**. It collects a host snapshot, evaluates explicit health policies, aggregates the resulting assessments, and reports the outcome through human-readable or structured JSON output.

V1 focuses on reliable host observation, explicit health semantics, transparent failure handling, and a clear separation between Linux collection and application behavior.

Continuous monitoring, exporters, dashboards, orchestration integrations, remote collection, and automatic remediation remain outside the current V1 scope.

---

## Table of Contents

* [Objective](#objective)
* [Current State](#current-state)
* [Quick Start](#quick-start)
* [V1 Application](#v1-application)
* [Human-Readable Output](#human-readable-output)
* [JSON Output](#json-output)
* [Exit Codes](#exit-codes)
* [Architecture](#architecture)
* [Host Snapshot](#host-snapshot)
* [Health Evaluation](#health-evaluation)
* [Memory Health](#memory-health)
* [Filesystem Health](#filesystem-health)
* [Process Health](#process-health)
* [Network Health](#network-health)
* [CPU Accounting and Utilization](#cpu-accounting-and-utilization)
* [Linux Collection Interfaces](#linux-collection-interfaces)
* [Experiments](#experiments)
* [Testing and Validation](#testing-and-validation)
* [Repository Structure](#repository-structure)
* [Failure Handling and Environmental Differences](#failure-handling-and-environmental-differences)
* [Documentation](#documentation)
* [Engineering Approach](#engineering-approach)
* [Design Principles](#design-principles)
* [Current V1 Scope](#current-v1-scope)
* [Deliberately Out of Scope for V1](#deliberately-out-of-scope-for-v1)
* [Configuration](#configuration)
* [Recommended First Run](#recommended-first-run)
* [Development Workflow](#development-workflow)
* [V1 Completion Criteria](#v1-completion-criteria)
* [Project Direction](#project-direction)
* [Philosophy](#philosophy)

---

## Objective

Build a small, technically defensible Linux host inspection and health agent that can answer reliability questions using observations from the operating system.

The project has two related objectives:

1. Understand the Linux interfaces that expose host state.
2. Encode that understanding into a reliable, testable, and explainable system.

hostcheck separates observation, derivation, policy, assessment, aggregation, and presentation.

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

Each stage has a distinct responsibility.

Collectors report what Linux exposes. Derived calculations interpret raw measurements. Health policies define acceptable operating conditions. Assessments explain the decisions made by those policies. Aggregation combines the assessments into a host-level result, while the application presents that result to an operator or another program.

This separation prevents raw operating-system measurements from becoming coupled to arbitrary thresholds or presentation logic.

## Current State

hostcheck has a working one-shot command-line application backed by Linux subsystem collectors and a separate health evaluation layer.

The current implementation covers five major areas of host observation:

* CPU accounting.
* System memory.
* Filesystem capacity and inode availability.
* Process statistics, CPU accounting, and memory accounting.
* Network interfaces, addresses, routes, and multipath route information.

The host model combines subsystem observations into a common snapshot. The health layer evaluates supported policies and produces assessments containing availability, status, reasons, and supporting evidence.

The application supports human-readable output and structured JSON, along with process exit codes for shell automation.

The human-readable report includes:

* Observation timestamp.
* Host-level status and coverage.
* Collection results.
* CPU accounting counters.
* Memory statistics and derived capacity.
* Filesystem capacity and inode information.
* Process totals and state information.
* Network interfaces, addresses, and routes.
* Health assessments with explanations and evidence.

CPU utilization is explicitly identified as not evaluated in the single-snapshot path because meaningful utilization requires a second observation.

The repository also contains subsystem tests, controlled Linux experiments, and documentation describing accounting behavior and implementation decisions.

The current application flow is:

```text
Linux interfaces
       |
       v
   Collectors
       |
       v
 Host Snapshot
       |
       v
Assessment orchestration
       |
       v
  Aggregation
       |
       v
 Health Result
       |
       +----------------+
       |                |
       v                v
 Human-readable      JSON output
     output
```

The project is deliberately developed from the behavior of the underlying operating system rather than from assumptions inherited from generic monitoring frameworks.

---

## Quick Start

### Requirements

hostcheck currently requires:

* Linux.
* Go.
* Access to the Linux interfaces used by its collectors.

The project is Linux-specific because it relies on interfaces such as `/proc`, `statfs`, and rtnetlink.

Check your installed Go version:

```bash
go version
```

### Clone the repository

```bash
git clone https://github.com/noklash/hostcheck.git
cd hostcheck
```

### Run the tests

```bash
go test ./...
```

### Run static analysis

```bash
go vet ./...
```

### Build the packages

```bash
go build ./...
```

### Run the host health agent

Human-readable output:

```bash
go run ./cmd/hostcheck
```

Structured JSON output:

```bash
go run ./cmd/hostcheck --json
```

These commands execute the one-shot application against the Linux host on which they run.

---

## V1 Application

The application entry point is:

```text
cmd/hostcheck/
```

The command connects collection, host snapshot integration, health evaluation, aggregation, and output.

Its responsibility is to orchestrate the existing components and present the resulting health report. It should not duplicate Linux collection logic or independently implement the health model.

The application currently uses explicit default policies for:

* Available memory capacity.
* Available filesystem capacity.
* Available filesystem inodes.

The default thresholds are:

| Resource                      |  Degraded |  Critical |
| ----------------------------- | --------: | --------: |
| Available memory              | Below 20% | Below 10% |
| Available filesystem capacity | Below 20% | Below 10% |
| Available filesystem inodes   | Below 20% | Below 10% |

These thresholds are application defaults, not universal definitions of Linux health. Different workloads and operating environments may require different policies.

Process-state and network-interface or route policies are supported by the health layer, but they are not enabled by default in the current CLI policy. Those checks require explicit expectations about which processes, interfaces, and routes should exist on a particular host.

CPU utilization is also not evaluated from a single snapshot.

### Application responsibilities

The application should:

1. Collect the available host observations.
2. Preserve collection errors.
3. Construct the host snapshot.
4. Run the configured health evaluation.
5. Aggregate the assessments.
6. Produce a human-readable or JSON report.
7. Return an exit code consistent with the resulting host-level status.

The application consumes the health result instead of duplicating health evaluation and aggregation logic.

---

## Human-Readable Output

Run the application without the JSON flag:

```bash
go run ./cmd/hostcheck
```

The report presents the collected host information and the result of the configured health assessments.

A successful run reports the observation timestamp, host-level status, coverage, subsystem information, and assessment details.

The exact output and measurements depend on the current implementation and the host being inspected.

An assessment contains a status, an explanation, and evidence where applicable. This allows an operator to understand the basis of a health decision instead of receiving only a single status label.

For example, a memory assessment may communicate that available memory is within the configured policy and include the measured available-memory percentage.

The report also distinguishes collected CPU accounting counters from CPU utilization. The counters can be collected from one observation, but utilization requires multiple observations.

The human-readable report is intended for direct inspection and troubleshooting. Structured JSON provides the machine-readable representation.

---

## JSON Output

hostcheck supports structured JSON output:

```bash
go run ./cmd/hostcheck --json
```

The JSON representation exposes the host-level result and the individual assessments that produced it.

An illustrative assessment structure is:

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

This is an illustrative response, not a guaranteed output from every run. Actual measurements and timestamps depend on the host.

The JSON representation preserves the host-level status and coverage alongside assessment information, including:

* Observation timestamp.
* Assessment subject.
* Assessment availability.
* Assessment status when applicable.
* Reason for the assessment.
* Supporting evidence.

JSON provides a structured integration boundary without requiring the internal health model to depend on a particular presentation format.

An unassessable observation is represented explicitly. A collection or evaluation failure does not automatically become a critical health assessment.

The JSON output should be interpreted together with the exit code when the command is used by shell scripts or other programs.

---

## Exit Codes

The command provides process exit codes suitable for shell automation and future operational integration.

The current exit-code policy is:

| Result                     | Exit code |
| -------------------------- | --------: |
| OK                         |       `0` |
| Degraded                   |       `1` |
| Critical                   |       `2` |
| Unavailable                |       `1` |
| Invalid application result |       `1` |

Exit codes communicate the host-level result without replacing the detailed report.

A caller can use the exit code to determine whether the command returned an OK, degraded, critical, or unavailable outcome. Human-readable or JSON output provides the details needed to understand the decision.

The distinction between degraded and critical results allows callers to respond differently to resource conditions.

An unavailable result indicates that a usable host-level assessment could not be established.

---

## Architecture

hostcheck is organized around Linux observation, explicit data models, derived measurements, and health evaluation.

```text
                         Linux
                           |
           +---------------+---------------+
           |               |               |
           v               v               v
        /proc            statfs        rtnetlink
           |               |               |
           v               v               v
       Collectors      Collectors      Collectors
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
                      Aggregation
                           |
                           v
                     Health Result
                           |
                    +------+------+
                    |             |
                    v             v
             Human-readable     JSON
                 output        output
```

### Collection

Collection reads operating-system state and preserves the information needed by higher layers.

A collector should not decide whether an observation is healthy simply because it can read a value. Collection and policy evaluation have different responsibilities.

### Derivation

Derivation calculates measurements that require interpretation of raw values.

Examples include:

* CPU utilization from counter deltas.
* Available memory percentage.
* Available filesystem capacity percentage.
* Available inode percentage.
* Process CPU utilization.

Derived calculations remain separate from the raw observations on which they depend.

### Policy

Health policies define acceptable conditions for a particular operational context.

A policy might specify the minimum acceptable percentage of available memory or the expected operational state of a network interface.

Thresholds and expectations belong in the health layer rather than being hidden inside collectors.

### Assessment

An assessment represents an evaluation of a particular health condition.

The core model is:

```go
type Assessment struct {
    Subject      string
    Availability Availability
    Status       Status
    Reason       string
    Evidence     []string
}
```

The model separates whether a condition can be evaluated from the health status assigned to it.

The explanation and evidence make an assessment useful beyond its status alone.

### Aggregation

Aggregation combines individual assessments into a host-level result.

The result model is:

```go
type Result struct {
    Status      Status
    Coverage    Coverage
    Assessments []Assessment
}
```

Aggregation preserves both the overall status and the coverage of the evaluation.

### Application

The command-line application connects collection, evaluation, aggregation, and output.

The application layer should remain thin. Linux-specific collection behavior belongs in the relevant subsystem packages, while health semantics belong in the health layer.

---

## Host Snapshot

The host integration boundary is represented by `host.Snapshot`.

Its current model includes:

```go
type Snapshot struct {
    ObservedAt time.Time
    CPU        *cpu.Stat
    Memory     *memory.MemInfo
    Filesystem *filesystem.Stats
    Processes  []process.Process
    Network    *network.Network
    Errors     []CollectionError
}
```

The snapshot brings subsystem observations together without requiring every subsystem to share the same collection mechanism.

Collection failures are represented explicitly:

```go
type CollectionError struct {
    Subsystem string
    Err       error
}
```

This allows the host model to preserve information about which subsystem failed and why.

### Snapshot semantics

Collection is sequential in the current implementation. The host timestamp represents the snapshot's observation boundary, not a claim that every subsystem was measured at precisely the same instant.

Linux continues changing while it is being inspected. Processes can exit, counters can advance, network interfaces can change state, and routes can be updated between individual reads.

A host snapshot is therefore a best-effort representation of the host during the collection process.

The model should preserve that distinction rather than imply atomicity that the underlying interfaces do not provide.

---

## Health Evaluation

Health evaluation is separate from Linux collection.

The health layer supports policies for:

* Available memory capacity.
* Available filesystem capacity.
* Available filesystem inode capacity.
* Process state.
* Network-interface state.
* Network routes.
* Host-level assessment aggregation.

CPU utilization assessment is available as a separate capability because it requires multiple observations.

### Availability

An assessment can be either:

```text
assessable
```

or:

```text
unassessable
```

An assessable observation contains sufficient information to evaluate the corresponding policy.

An unassessable observation cannot be evaluated reliably with the information available.

For example, a missing measurement or failed collection may prevent a policy from being evaluated.

The distinction prevents the system from treating every observation failure as proof that the host is unhealthy.

An unassessable assessment does not receive an ordinary health status. Its availability communicates that the evaluation could not be completed.

### Health status

Assessable observations can have one of three statuses:

```text
ok
degraded
critical
```

The severity ordering is:

```text
ok < degraded < critical
```

Critical assessments take precedence over degraded assessments, and degraded assessments take precedence over OK assessments when determining severity.

Unassessable assessments do not automatically become critical.

### Coverage

The host-level result reports how much of the supplied assessment set could be evaluated.

The coverage values are:

| Coverage      | Meaning                                                                   |
| ------------- | ------------------------------------------------------------------------- |
| `complete`    | All supplied assessments were assessable.                                 |
| `partial`     | At least one assessment was assessable and at least one was unassessable. |
| `unavailable` | No supplied assessment was assessable.                                    |

Coverage and status answer different questions.

Status describes the evaluated health conditions. Coverage describes how much of the assessment set could be evaluated.

For example, a critical result with partial coverage indicates that at least one evaluated condition is critical while another condition could not be assessed.

That is materially different from a critical result with complete coverage.

### Assessment evidence

A status alone is insufficient for an explainable health system.

Consider the difference between:

```text
status: degraded
```

and:

```text
status: degraded
reason: available memory capacity is below the configured degraded threshold
evidence: available_percent=15.25
```

The second form provides an explanation and a measurement supporting the decision.

hostcheck preserves reasons and evidence so the operator can understand what the policy evaluated.

---

## Memory Health

Memory health evaluates available memory capacity using Linux's `MemAvailable` value rather than `MemFree`.

The derived observation is:

```text
available_percent = MemAvailable / MemTotal * 100
```

`MemAvailable` estimates memory that can be made available to applications without requiring the system to begin swapping. It accounts for more than immediately unused physical memory.

`MemFree` reports a different quantity and should not be substituted for available memory when evaluating the same policy.

The default application thresholds are:

```text
degraded: below 20%
critical: below 10%
```

These are explicit application defaults rather than universal Linux health thresholds.

The health policy evaluates available memory capacity at the time of observation.

It does not establish:

* Historical memory pressure.
* Memory pressure duration.
* Future out-of-memory conditions.
* Memory reclaim trends.
* Long-term swap behavior.

Those questions require additional measurements, repeated observations, or time-series information.

### Memory collection

System memory statistics are collected from:

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

The parser validates required fields, numeric values, units, duplicate fields, and missing fields.

The reader-based parsing boundary separates filesystem access from parsing, making the parser easier to test with controlled input.

Values are retained in the units reported by Linux. The memory model and derived calculations must preserve those units consistently.

Memory accounting details are documented in:

```text
docs/memory-accounting.md
```

The project has also used a controlled 512 MiB memory-pressure experiment to investigate changes in available memory, free memory, cache, and swap under workload.

---

## Filesystem Health

Filesystem health evaluates two distinct resource constraints:

* Available filesystem capacity.
* Available filesystem inodes.

Both use explicit percentage thresholds.

The default application policy is:

```text
degraded: below 20%
critical: below 10%
```

Capacity and inode availability are evaluated separately because a filesystem can have substantial free space in bytes while running out of inodes.

These conditions affect different operations and should not be represented by one combined measurement.

### Filesystem collection

Filesystem statistics are collected using Linux `statfs`.

The current model records:

* Filesystem path.
* Block size.
* Total blocks.
* Free blocks.
* Blocks available to an unprivileged process.
* Total inodes.
* Free inodes.

Derived values include:

* Used blocks.
* Used bytes.
* Available bytes.
* Available capacity percentage.
* Used inodes.
* Available inode percentage.

The implementation preserves the distinction between free blocks and blocks available to an unprivileged process. Linux exposes both because filesystem capacity and user-available capacity are not always identical.

Derived arithmetic is separated from raw collection. The implementation checks relevant arithmetic assumptions and overflow conditions before returning derived values.

Filesystem accounting details are documented in:

```text
docs/filesystem-capacity.md
```

### Filesystem health boundaries

Filesystem health evaluates the measurements supplied to its policies.

It does not automatically establish:

* Whether an application can write to every directory.
* Whether a particular user's quota has been reached.
* Whether a mount will remain available.
* Whether filesystem I/O latency is acceptable.
* Whether filesystem corruption is present.

Those conditions require additional observations and explicit policies.

---

## Process Health

The process subsystem collects observations from the live `/proc` filesystem.

The general collection flow is:

```text
/proc
  |
  v
Enumerate PIDs
  |
  v
Read /proc/<pid>/stat
  |
  v
Read /proc/<pid>/status
  |
  v
Construct Process
```

The subsystem supports:

* Process enumeration.
* `/proc/<pid>/stat` parsing.
* Process lifecycle race handling.
* Process CPU accounting.
* Process CPU utilization.
* Process memory accounting.
* Kernel-thread identification.
* Process-state observations.

### Process lifecycle races

A process can exit between PID enumeration and an individual `/proc` read. This is normal behavior on a live Linux system.

hostcheck accounts for this lifecycle race rather than assuming every enumerated PID will remain available throughout collection.

Processes that disappear during collection are handled as expected lifecycle events, where appropriate, instead of causing the entire collection to fail.

### Process CPU accounting

Process CPU accounting uses cumulative CPU-time values exposed through `/proc/<pid>/stat`.

Process CPU utilization requires multiple observations. A single cumulative counter establishes the amount of CPU time accounted to a process, not its utilization over an interval.

### Process memory accounting

The process model preserves information needed for process memory analysis, including virtual size and resident-set accounting.

Kernel threads are included in process collection and identified explicitly. Userspace memory information is not necessarily available for kernel threads in the same way it is for ordinary userspace processes.

Process memory details are documented in:

```text
docs/process-memory-accounting.md
```

Controlled experiments investigate process memory behavior under known workloads.

### Process-state policies

The health layer can evaluate selected process-state conditions.

For example, a process observed in Linux state `D` was in uninterruptible sleep when it was observed.

One observation does not establish how long the process has been in that state, whether the condition is persistent, or whether the process is permanently blocked.

Process-state health is therefore limited to the evidence available from the observation. Process-state policies are supported but are not enabled by the default CLI policy.

---

## Network Health

The network subsystem collects Linux interface information and kernel routing information.

The current network snapshot includes:

* Interface index.
* Interface name.
* Hardware address.
* Operational state.
* Carrier state.
* MTU.
* Optional link speed.
* Optional duplex information.
* Interface addresses.
* Routes.
* Multipath route next hops.

Network health supports explicit policies for:

* Expected interfaces.
* Interface operational state.
* Optional carrier requirements.
* Expected routes.
* Route interface selection.
* Route attributes.
* Multipath route expectations.

These policies evaluate local network configuration and observed interface state.

They do not establish gateway reachability, DNS availability, Internet connectivity, remote service reachability, or application-layer connectivity.

Those are separate questions that require an explicit active-probe model.

### Network addresses

Interface addresses are collected through Linux networking interfaces and rtnetlink.

An address retains its IP address and prefix length, together with Linux address-scope information where provided by the collector.

The implementation does not infer Linux address scope from Go's generic IP classification. Linux networking semantics should come from Linux networking data.

The collector also avoids treating parsed human-readable `ip` command output as its primary data source.

Address behavior is documented in:

```text
docs/network-addresses.md
```

### Network routes

Routes are collected through rtnetlink.

The route model preserves kernel-level attributes, including:

* Address family.
* Destination and destination prefix length.
* Source and source prefix length.
* Gateway.
* Output interface index.
* Priority.
* Routing table.
* Protocol.
* Scope.
* Route type.
* Route flags.
* Multipath next hops.

Linux routing cannot always be reduced to a single default gateway. Systems may use multiple routing tables, different route priorities, source-based routing, multiple protocols, and multipath routes.

Preserving those attributes gives higher layers access to the underlying routing structure instead of flattening it into an oversimplified model.

Routing behavior is documented in:

```text
docs/network-routing.md
```

### Multipath routes

Linux routes can contain multiple next hops.

hostcheck preserves multipath information explicitly, including the interface index, gateway, hop information, and flags associated with each next hop.

This avoids losing routing information by representing a multipath route as if it had only one gateway or interface.

### Network reachability boundary

The current network health model can evaluate conditions such as:

```text
Expected interface exists
Expected interface is operational
Expected route exists
Expected route uses the expected interface
```

These observations do not prove:

```text
Gateway is reachable
DNS is working
Internet is reachable
Remote API is reachable
Application is healthy
```

An active reachability subsystem would need defined destinations, protocols, timeouts, retries, failure classifications, and network-namespace behavior.

Active network reachability probes are outside the current V1 scope.

---

## CPU Accounting and Utilization

CPU accounting is collected from:

```text
/proc/stat
```

The aggregate Linux CPU record contains counters for:

```text
user nice system idle iowait irq softirq steal guest guest_nice
```

These values represent cumulative CPU time accounted by Linux in clock ticks.

hostcheck preserves the raw counters and derives utilization from changes between observations.

The model distinguishes:

* User time.
* Nice time.
* System time.
* Idle time.
* I/O wait.
* Hardware interrupt time.
* Software interrupt time.
* Steal time.
* Guest time.
* Guest-nice time.

Guest time is retained as raw information. It is not independently added to the total because Linux already accounts for guest time within other CPU counters.

### CPU utilization

CPU utilization is derived from counter deltas:

```text
delta_total = total_after - total_before

delta_busy = busy_after - busy_before

utilization = delta_busy / delta_total
```

The implementation must use consistent accounting semantics when calculating total and busy time.

It also detects counter regression. If a cumulative counter moves backwards, the derived calculation treats the input as invalid rather than silently producing an incorrect utilization value.

### Sampling boundary

CPU utilization requires at least two observations separated by time.

```text
Sample 1
   |
   | Time passes
   v
Sample 2
   |
   v
Counter deltas
   |
   v
CPU utilization
```

A one-shot snapshot can collect CPU counters, but it cannot derive meaningful interval utilization from a single sample.

For that reason, CPU utilization remains separate from the current single-snapshot health evaluation path.

A future sampling or continuous-evaluation layer can build on this model without changing the meaning of the underlying CPU observations.

---

## Linux Collection Interfaces

hostcheck works directly with Linux operating-system interfaces.

| Interface                  | Purpose                                                              |
| -------------------------- | -------------------------------------------------------------------- |
| `/proc/stat`               | Aggregate CPU accounting                                             |
| `/proc/meminfo`            | System memory accounting                                             |
| `/proc/<pid>/stat`         | Process statistics and CPU accounting                                |
| `/proc/<pid>/status`       | Process state and memory-related information                         |
| `statfs`                   | Filesystem capacity and inode accounting                             |
| rtnetlink                  | Network addresses, routes, and related kernel networking information |
| Network interface metadata | Interface identity and link information                              |

These interfaces remain visible in the implementation.

Abstractions are introduced where they improve testability, separation of concerns, or integration boundaries. They are not introduced merely to hide the behavior of Linux.

The collectors are designed around operating-system interfaces rather than around the output of human-readable shell commands.

During investigation, tools such as `cat`, `df`, and `ip` remain useful for comparing collector observations against the system's own diagnostic output.

---

## Experiments

Experiments are small programs used to investigate Linux behavior before that behavior is encoded into production collectors.

Current experiment areas include:

```text
experiments/
├── cpu/
├── filesystem/
├── network-addresses/
├── network-route-table/
├── network-routing/
├── network-rtnetlink/
├── process/
├── process-cpu/
└── process-memory/
```

Run an experiment with:

```bash
go run ./experiments/<experiment>
```

For example:

```bash
go run ./experiments/cpu
```

### CPU experiment

Location:

```text
experiments/cpu/
```

Investigates:

* `/proc/stat`.
* Cumulative CPU counters.
* Counter deltas.
* Aggregate utilization.
* Sampling behavior.

### Filesystem experiment

Location:

```text
experiments/filesystem/
```

Investigates:

* Filesystem capacity.
* Block accounting.
* Inode accounting.
* Linux filesystem statistics.

### Process experiment

Location:

```text
experiments/process/
```

Investigates:

* Process enumeration.
* `/proc/<pid>/stat`.
* `/proc/<pid>/status`.
* Process lifecycle behavior.

### Process CPU experiment

Location:

```text
experiments/process-cpu/
```

Investigates process CPU accounting and utilization across multiple observations.

### Process memory experiment

Location:

```text
experiments/process-memory/
```

Uses controlled memory mappings and workload phases to investigate how process memory measurements appear through Linux `/proc` interfaces.

### Network address experiment

Location:

```text
experiments/network-addresses/
```

Investigates interface addresses and their representation through Linux networking interfaces and rtnetlink.

### Network routing experiments

Locations:

```text
experiments/network-routing/
experiments/network-route-table/
```

Investigate routing information, route tables, route attributes, and IPv4 and IPv6 routing behavior.

### Network rtnetlink experiment

Location:

```text
experiments/network-rtnetlink/
```

Inspects rtnetlink information before it is represented by the production network collector.

### Why experiments matter

Experiments are part of the engineering process. They establish what Linux actually reports before the production implementation assigns meaning to those observations.

The workflow is:

```text
Question
   |
   v
Inspect Linux behavior
   |
   v
Design controlled experiment
   |
   v
Record observations
   |
   v
Define the model
   |
   v
Implement
   |
   v
Test and document
```

This approach is particularly important for CPU accounting, memory accounting, process memory, process lifecycle behavior, filesystem capacity, network routing, and rtnetlink.

---

## Testing and Validation

Testing is performed alongside implementation.

The repository's tests cover areas including:

* Valid parsing.
* Malformed input.
* Invalid numeric values.
* Invalid units.
* Missing fields.
* Duplicate fields.
* CPU counter regression.
* CPU utilization calculations.
* Filesystem arithmetic.
* Filesystem overflow.
* Filesystem collector behavior.
* Process enumeration.
* Process lifecycle races.
* Process CPU calculations.
* Process memory parsing.
* Kernel-thread behavior.
* Network interface collection.
* Network address parsing.
* Route parsing.
* Multipath route handling.
* Collector failures.
* Reader failures.
* Health assessment validation.
* Health aggregation.
* Memory health policies.
* Filesystem health policies.
* Inode health policies.
* Process-state health.
* Network-interface health.
* Network-route health.
* Host snapshot evaluation.
* Application policy defaults.
* Application exit codes.
* JSON serialization.
* Unavailable JSON results.

Run the complete test suite:

```bash
go test ./...
```

Run static analysis:

```bash
go vet ./...
```

Build all packages:

```bash
go build ./...
```

Format Go source files:

```bash
gofmt -w .
```

Check for whitespace errors:

```bash
git diff --check
```

For changes to the application boundary, also run both output modes:

```bash
go run ./cmd/hostcheck
go run ./cmd/hostcheck --json
```

The human-readable and JSON representations should describe the same underlying health result.

### Development validation sequence

The normal validation sequence is:

```bash
gofmt -w .
go test ./...
go vet ./...
go build ./...
git diff --check
git status --short --branch
```

These commands provide complementary checks for formatting, correctness, static issues, build failures, whitespace errors, and repository state.

---

## Repository Structure

The repository is organized around Linux subsystems and engineering boundaries.

A simplified view of the project is:

```text
hostcheck/
├── cmd/
│   └── hostcheck/
├── internal/
│   ├── cpu/
│   ├── filesystem/
│   ├── health/
│   ├── host/
│   ├── memory/
│   ├── network/
│   └── process/
├── docs/
│   ├── filesystem-capacity.md
│   ├── health-evaluation.md
│   ├── memory-accounting.md
│   ├── network-addresses.md
│   ├── network-routing.md
│   └── process-memory-accounting.md
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
├── go.mod
└── README.md
```

The exact file layout may evolve as the implementation develops.

The main structural boundaries are:

| Directory              | Responsibility                                           |
| ---------------------- | -------------------------------------------------------- |
| `cmd/`                 | Application entry points and CLI behavior                |
| `internal/cpu/`        | CPU collection and accounting                            |
| `internal/memory/`     | System memory collection and derived memory measurements |
| `internal/filesystem/` | Filesystem capacity and inode accounting                 |
| `internal/process/`    | Process collection, accounting, and memory observations  |
| `internal/network/`    | Network interfaces, addresses, and routes                |
| `internal/host/`       | Host-level snapshot integration                          |
| `internal/health/`     | Health policies, assessments, and aggregation            |
| `experiments/`         | Controlled investigation of Linux behavior               |
| `docs/`                | Subsystem documentation and engineering decisions        |

The package structure keeps collection, host integration, health evaluation, and application behavior understandable without requiring a larger application framework.

The implementation uses Go's `internal/` package boundary. These packages are designed for use within the repository rather than as a public Go library API for arbitrary external imports.

---

## Failure Handling and Environmental Differences

hostcheck is designed to distinguish between an unhealthy host and a host that could not be fully observed.

A collection failure may prevent an assessment from being evaluated. The health model preserves that limitation rather than automatically assigning a critical status.

This distinction matters because Linux behavior can vary with:

* Kernel configuration.
* `/proc` mount options.
* Process visibility restrictions.
* User permissions.
* Network namespaces.
* Containerization.
* Security policies.
* Filesystem configuration.
* Interface availability.

Most collectors can use information available to ordinary Linux users, but not every environment exposes the same information.

A process can disappear during enumeration. A network namespace can expose a different interface and route set. Security restrictions can make an otherwise valid observation unavailable.

The implementation should preserve these limitations explicitly rather than inventing values or presenting incomplete observations as complete.

The general rule is:

```text
Observation unavailable
          |
          v
Preserve the limitation
          |
          v
Report assessment coverage honestly
```

The host health result should describe what was actually observed and evaluated.

---

## Documentation

Subsystem-specific documentation records the Linux behavior and design decisions behind the implementation.

Current documentation includes:

```text
docs/memory-accounting.md
docs/filesystem-capacity.md
docs/process-memory-accounting.md
docs/network-addresses.md
docs/network-routing.md
docs/health-evaluation.md
```

The documentation covers relevant accounting semantics, implementation decisions, assumptions, experiments, limitations, and health evaluation behavior.

The README provides the project-level view. Detailed subsystem behavior belongs in the corresponding documentation.

This division keeps the main documentation useful as an architectural overview without requiring it to reproduce every implementation detail.

Documentation should remain consistent with the actual implementation. When a subsystem or application behavior changes, the relevant tests and documentation should be reviewed alongside the code.

---

## Engineering Approach

Each subsystem follows the same general engineering workflow:

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

The workflow establishes the meaning of an operating-system observation before that observation becomes an assumption in production code.

For each subsystem, the important questions include:

1. What does Linux actually expose?
2. What does each field mean?
3. Which values are cumulative counters?
4. Which values are derived?
5. What can change during collection?
6. Which failure modes are normal?
7. Which assumptions are safe to encode?
8. Which information should remain raw?
9. What belongs to collection?
10. What belongs to derivation?
11. What belongs to health policy?
12. What can the available observations actually establish?

These questions keep the implementation grounded in operating-system behavior.

Experiments provide evidence. Tests protect established behavior. Documentation records the reasoning. Small, logically focused commits preserve the implementation history.

The repository history should show how the system was investigated, implemented, tested, and refined, rather than only presenting the final code.

---

## Design Principles

hostcheck favors:

* Linux-native interfaces.
* Small, focused packages.
* Explicit data models.
* Deterministic tests.
* Controlled experiments.
* Clear failure handling.
* Documented assumptions.
* Raw observations before interpretation.
* Explicit health policies.
* Explainable assessments.
* Honest coverage reporting.
* Small logical commits.
* Readable implementation.
* Deliberate abstraction.

The project avoids:

* Framework-heavy architecture.
* Premature distributed-system design.
* Hidden or arbitrary health thresholds.
* Unnecessary abstraction.
* Parsing human-readable shell output as the primary collection mechanism.
* Infrastructure integrations before the host model is understood.
* Active network probes without a defined probe model.
* Treating unavailable observations as automatic failures.
* Claiming that a single observation proves a persistent condition.

The objective is to keep the system small enough to understand while making its behavior technically defensible.

---

## Current V1 Scope

The current V1 scope includes the following capabilities.

### Linux observation

* CPU accounting.
* System memory accounting.
* Filesystem capacity and inode accounting.
* Process enumeration and statistics.
* Process CPU and memory accounting.
* Process-state observations.
* Network interface state.
* Network addresses.
* Network routes.
* Multipath route representation.

### Derived measurements

* CPU utilization from counter deltas.
* Available memory capacity percentage.
* Filesystem capacity percentages.
* Available inode percentage.
* Process CPU utilization.
* Filesystem usage calculations.

### Host integration

* Host snapshots.
* Observation timestamps.
* Collection error representation.
* Integration of subsystem observations.

### Health evaluation

* Memory capacity policies.
* Filesystem capacity policies.
* Inode availability policies.
* Process-state policies.
* Network-interface policies.
* Network-route policies.
* Assessment availability.
* Health status.
* Assessment evidence.
* Host-level aggregation.
* Coverage reporting.

### Application

* One-shot CLI execution.
* Human-readable output.
* Structured JSON output.
* Process exit codes.
* Default health policies.

### Engineering evidence

* Deterministic tests.
* Controlled Linux experiments.
* Subsystem documentation.
* Explicit architectural boundaries.

The current V1 focuses on the reliability of these existing layers rather than expanding into a larger monitoring platform.

---

## Deliberately Out of Scope for V1

The following capabilities are not currently part of the V1 application:

* Daemon mode.
* Continuous background monitoring.
* Prometheus exporters.
* Grafana dashboards.
* Kubernetes integration.
* Docker integration.
* Cloud-provider integrations.
* Remote collection.
* Distributed collection.
* Persistent metric storage.
* Alerting infrastructure.
* Automatic remediation.
* Active network reachability probes.
* DNS health probes.
* Internet reachability checks.
* Application-level service probes.

These capabilities may become useful as the project develops, but they are not prerequisites for establishing a reliable host observation and health evaluation model.

A future implementation should build on the existing boundaries rather than introduce infrastructure before its requirements are clear.

---

## Configuration

The application currently uses explicit built-in default policies for the basic V1 health checks.

It does not yet introduce a large configuration system.

Configuration becomes useful when there is a concrete operational need to customize:

* Resource thresholds.
* Expected interfaces.
* Expected routes.
* Process-state policies.
* Output behavior.

The configuration model should follow those requirements rather than being introduced simply because other monitoring tools have configuration files.

The current application boundary remains deliberately small.

---

## Recommended First Run

For someone exploring the repository for the first time:

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

Read the subsystem documentation:

```text
docs/memory-accounting.md
docs/filesystem-capacity.md
docs/process-memory-accounting.md
docs/network-addresses.md
docs/network-routing.md
docs/health-evaluation.md
```

A useful exploration sequence is:

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

This sequence connects operating-system behavior to the implementation and then to the resulting health assessment.

---

## Development Workflow

Before committing changes, run:

```bash
gofmt -w .
go test ./...
go vet ./...
go build ./...
git diff --check
git status --short --branch
```

For changes involving the application boundary, also run:

```bash
go run ./cmd/hostcheck
go run ./cmd/hostcheck --json
```

Commits should represent one logical engineering change.

Examples include:

```text
network: collect addresses and routes via rtnetlink
memory: derive available capacity
health: aggregate host assessments
cmd: add host health check entrypoint
cmd: add JSON output
cli: improve host health report output
docs: reconcile README with V1
```

The repository history is intended to show how the system was investigated and built, not merely what the final code looks like.

Validation results should be recorded accurately. A successful test run establishes that the tested code passed that run; it does not establish that every possible Linux environment has been tested.

---

## V1 Completion Criteria

The first usable V1 should be judged by engineering behavior and clarity rather than by the number of features.

The core criteria are:

* Linux observations are collected correctly.
* Derived values have explicit semantics.
* Collection failures are represented honestly.
* Health policies are explicit.
* Assessments explain their decisions.
* Aggregation preserves coverage information.
* The application can perform a one-shot evaluation.
* Human-readable output is useful to an operator.
* JSON output is usable by another program.
* Exit codes communicate the host-level result.
* Tests cover important failure modes.
* Documentation reflects the actual implementation.
* The architecture remains understandable without a larger framework.

A V1 review should also verify that:

* Both output modes handle the same underlying health result.
* Unassessable conditions are represented honestly.
* The collection and evaluation boundaries remain separate.
* CPU utilization is not presented as a single-snapshot measurement.
* Network configuration is not presented as proof of end-to-end connectivity.
* The full test suite passes.
* Static analysis passes.
* The repository builds successfully.
* The working tree and Git history are understood before a release or handoff.

The goal is a small host health system whose behavior can be explained from the Linux interfaces upward.

---

## Project Direction

hostcheck is being developed as a systems engineering project. The direction is to move upward through the stack only after each underlying layer is understood.

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

The current V1 application boundary is intentionally small:

```text
Collect
  |
  v
Snapshot
  |
  v
Evaluate
  |
  v
Aggregate
  |
  v
Report
```

The next stage of development should strengthen the existing boundaries, verify the documentation against the implementation, and close any remaining V1 gaps before adding a daemon, exporter, distributed agent, or orchestration platform.

The long-term value of the project lies in understanding the operating system, preserving its semantics, and building reliable software around that understanding.

---

## Philosophy

hostcheck is intended to make Linux behavior understandable first, then encode that understanding into a reliable tool.

The project moves from:

```text
Kernel interface
       |
       v
Observation
       |
       v
Measurement
       |
       v
Policy
       |
       v
Health decision
       |
       v
Operational result
```

Each stage should be explicit, testable, and explainable.

> First understand the system. Then master the tools. Finally, build the platform.
