
# hostcheck

A Linux host health and reliability agent written in Go.

hostcheck reads operating-system interfaces exposed by Linux and turns them into structured host information. The project is being built from the kernel interfaces upward, with each subsystem researched, implemented, tested, and documented before moving to the next.

The initial version is intentionally a one-shot collector. Continuous monitoring, exporters, dashboards, orchestration integrations, and other operational layers are outside the current scope.

## Objective

Build a small but technically defensible Linux host inspection tool that can answer basic reliability questions directly from the operating system.

The project is also an engineering study of the Linux interfaces behind those answers.

The guiding principle is:

> Understand the system first. Then build the tool.

## Current State

The project currently implements CPU and memory collection.

### CPU

CPU accounting is collected from:

```text
/proc/stat
````

The implementation currently supports:

* parsing aggregate CPU accounting,
* cumulative CPU counter deltas,
* counter regression detection,
* aggregate CPU utilization,
* separate treatment of I/O wait and steal time,
* preservation of guest and guest-nice counters,
* direct collection from `/proc/stat`,
* deterministic parser and collector tests.

CPU utilization is calculated from counter deltas rather than assuming that a requested sleep duration represents the exact sampling interval.

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

A controlled 512 MiB memory-pressure experiment was also performed to understand how available memory, cache, and swap changed under workload.

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

Higher layers will be responsible for interpretation, health decisions, and presentation.

This separation is deliberate. Raw kernel measurements should not be mixed with policy or health thresholds.

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
│   │   └── utilization.go
│   │
│   └── memory/
│       ├── collector.go
│       ├── collector_test.go
│       ├── meminfo.go
│       ├── parser.go
│       └── parser_test.go
│
├── experiments/
│   └── cpu/
│       ├── main.go
│       └── README.md
│
├── docs/
│   └── memory-accounting.md
│
├── go.mod
└── README.md
```

The repository is intentionally organized around subsystems rather than around a large application framework.

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

The utilization calculation is based on the change in these counters between two samples.

## Memory Collection

The memory collector follows:

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

The parser ignores fields that are outside the current V1 model while validating all required fields.

Values are retained in the `kB` unit reported by Linux.

## Experiments

The project uses small controlled experiments to validate understanding before encoding behavior into the collector.

Current experiments include:

### CPU accounting

```text
experiments/cpu/
```

The experiment samples `/proc/stat`, calculates counter deltas, and derives aggregate CPU utilization.

The experiment also records the system configuration and observed behavior.

### Memory pressure

A 512 MiB allocation was used to observe changes in:

* available memory,
* free memory,
* cache,
* swap usage.

The experiment showed why `MemFree` alone is insufficient as a memory-health signal and why the kernel-provided `MemAvailable` value is important.

## Testing

Tests are written alongside each subsystem.

The project currently tests:

* valid parsing,
* malformed input,
* invalid numeric values,
* invalid units,
* missing fields,
* duplicate fields,
* CPU counter regressions,
* CPU utilization calculation,
* collector behavior,
* reader failures.

Run the full test suite with:

```bash
go test ./...
```

Run static analysis with:

```bash
go vet ./...
```

Check for whitespace errors with:

```bash
git diff --check
```

Format Go source with:

```bash
gofmt -w .
```

## Engineering Approach

Each subsystem follows the same general sequence:

```text
Research
   |
   v
Observe Linux directly
   |
   v
Define the data model
   |
   v
Implement parser
   |
   v
Write parser tests
   |
   v
Implement collector
   |
   v
Write collector tests
   |
   v
Run controlled experiment
   |
   v
Document behavior and limitations
```

The purpose is to prevent the project from becoming a collection of copied metrics with unclear semantics.

For every metric, the project should be able to explain:

* where the value comes from,
* what the kernel reports,
* what the raw representation means,
* how the value is calculated,
* what assumptions are being made,
* what the value does not tell us.

## Scope

The current V1 scope is intentionally limited.

Included:

* Linux host inspection,
* direct kernel interfaces,
* CPU statistics,
* memory statistics,
* deterministic tests,
* controlled experiments,
* technical documentation.

Not currently included:

* daemon mode,
* Prometheus exporters,
* Grafana dashboards,
* Kubernetes integration,
* Docker integration,
* cloud-provider integrations,
* databases,
* remote monitoring,
* alerting infrastructure,
* distributed collection.

These may become relevant later, but they are not prerequisites for understanding and implementing the host-level collection layer.

## Planned Subsystems

The remaining host inspection work is expected to cover areas such as:

```text
CPU
Memory
Filesystem capacity
Inodes
Processes
Network
Health evaluation
Structured output
```

The exact design of each subsystem will be determined after studying the corresponding Linux interfaces.

In particular, health thresholds will not be introduced simply because a metric exists. A threshold needs an operational reason and defensible semantics.

## Design Constraints

### Linux first

The operating system is the primary source of truth.

Where possible, hostcheck reads the kernel interface directly instead of relying on another monitoring utility to produce the data.

### Raw collection before interpretation

Collectors should obtain reliable raw state.

Health policy belongs in a separate layer.

### Small interfaces

Each subsystem should have a small data model and clear responsibility.

### Deterministic tests

Tests should not depend on whatever happens to be occurring on the developer's machine at the time they run.

### Evidence over assumptions

Experiments are used where behavior is easier to understand by observation.

### Document limitations

A metric is only useful when its limitations are understood.

## Development Workflow

Changes are kept in small logical commits.

Typical progression:

```text
research
→ experiment
→ implementation
→ tests
→ documentation
→ validation
→ commit
```

Before committing Go changes:

```bash
gofmt -w .
go test ./...
go vet ./...
git diff --check
```

The Git history is intended to show the evolution of the engineering work rather than a single large implementation commit.

## Status

Current implementation:

```text
CPU
├── accounting model       complete
├── parser                 complete
├── parser tests           complete
├── counter deltas         complete
├── regression handling    complete
├── utilization            complete
├── collector              complete
├── collector tests        complete
└── experiment             complete

Memory
├── accounting research    complete
├── data model             complete
├── parser                 complete
├── parser tests           complete
├── collector              complete
├── collector tests        complete
└── documentation          complete
```

The next implementation phase will extend hostcheck to another Linux subsystem after the existing CPU and memory work has been reviewed and validated.

## License

License information has not yet been defined.

