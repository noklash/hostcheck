# Host Snapshot

Hostcheck collects system state through separate subsystem collectors. The host layer provides the boundary where those observations are assembled into a single host snapshot.

The snapshot is the input to future health evaluation and output layers. It does not decide whether the host is healthy.

## Purpose

The host snapshot gives Hostcheck one top-level representation of the state observed across its current subsystems:

* CPU
* memory
* filesystem
* processes
* network

Each subsystem remains responsible for understanding and collecting its own Linux data. The host layer is responsible for composition.

The current architecture is:

```text
Linux kernel interfaces
        ↓
Subsystem collectors
        ↓
Host snapshot
        ↓
Health evaluation
        ↓
Output
```

The health and output layers are intentionally separate from collection.


## Snapshot Model

The host snapshot is represented by `host.Snapshot`:

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

The snapshot contains pointers for subsystem observations that may fail independently.

Processes are represented as a slice because a successful process collection may legitimately return zero processes.

## Observation Time

`Snapshot.ObservedAt` records the time host collection begins.

It does not mean that every field in the snapshot was collected at exactly that instant.

Collection happens sequentially. CPU, memory, filesystem, processes, and network may therefore represent slightly different points in time.

The snapshot should be understood as:

> a collection of observations gathered during one host collection operation.

It is not an atomic system-wide snapshot.

The network subsystem also maintains its own `ObservedAt` timestamp because network collection has its own collection boundary.

## Partial Collection Failures

Hostcheck treats subsystem collection as best-effort.

A failure in one subsystem does not discard observations successfully collected from other subsystems.

For example, if process collection succeeds but network collection fails, the snapshot can still contain CPU, memory, filesystem, and process observations.

Failures are recorded in:

```go
type CollectionError struct {
    Subsystem string
    Err       error
}
```

This allows later layers to distinguish between:

* an observation that was successfully collected
* an observation that could not be collected
* a subsystem that legitimately returned an empty result

The host collector therefore returns a `Snapshot` directly rather than returning a separate top-level collection error.

## Nil and Empty Values

Pointer-based subsystem fields use `nil` to indicate that collection did not produce an observation.

For example:

```go
if snapshot.Memory == nil {
    // memory collection failed
}
```

A non-nil value means the subsystem produced an observation, even if some values inside that observation are empty or zero.

Processes use a slice:

```go
Processes []process.Process
```

The distinction is:

```text
nil slice
    ↓
process collection did not produce a successful result

empty non-nil slice
    ↓
process collection succeeded and found no processes
```

This distinction becomes useful when later layers evaluate collection completeness.

## Filesystem Scope

The filesystem collector requires a path.

For the V1 host snapshot, an empty filesystem path defaults to:

```text
/
```

This gives the host snapshot a stable default representation of the primary filesystem without introducing configuration complexity at this stage.

The underlying filesystem collector remains reusable for arbitrary paths.

## Network Observation

Network collection is currently treated as one subsystem from the host layer.

The network snapshot contains:

* its own observation timestamp
* interface identity
* interface state
* interface addresses
* routing information

The network collector currently gathers these observations through Linux interfaces including rtnetlink.

Network statistics are not currently part of the assembled `Network` snapshot.

The host layer does not reinterpret or normalize these observations. That remains the responsibility of the network subsystem.

## Collection Order

The current host collector performs subsystem collection sequentially:

```text
CPU
 ↓
Memory
 ↓
Filesystem
 ↓
Processes
 ↓
Network
```

This is intentional for the current V1 implementation.

Parallel collection has not been introduced because the primary goal at this stage is to establish clear subsystem boundaries and snapshot semantics.

If concurrency becomes necessary later, it should be introduced with explicit consideration for timestamps, resource contention, error handling, and testability.

## Health Policy

The host snapshot does not contain health thresholds or health decisions.

For example, the snapshot may contain:

```text
CPU utilization: 87%
```

It does not decide whether 87% represents:

* healthy
* warning
* critical

Those decisions belong to the health evaluation layer.

This separation keeps collection focused on answering:

> What did we observe?

while health evaluation can later answer:

> What does the observation mean?

This also prevents collection code from becoming coupled to a particular policy.

## Live-System Races

Hostcheck operates against a live Linux system.

Processes and other system state can change while collection is running. A process may disappear between enumeration and reading its `/proc` files, for example.

The subsystem collectors already account for expected live-system races where appropriate.

The host snapshot therefore represents observations from a changing system rather than attempting to freeze the machine during collection.

## Error Semantics

The host layer currently records subsystem errors without converting them into health states.

For example:

```text
network collection failed
```

remains a collection error.

It does not automatically become:

```text
network unhealthy
```

That distinction is important because an inability to observe a subsystem is different from observing a subsystem in an unhealthy state.

Future health evaluation can decide how collection errors should affect overall health.

## Current Limitations

The V1 host snapshot intentionally leaves several concerns for later work:

* health thresholds
* health status evaluation
* structured output
* JSON serialization
* CLI presentation
* collection duration
* richer error metadata
* concurrent subsystem collection
* atomicity guarantees
* configurable filesystem scope
* network statistics integration

These are deliberately deferred until the snapshot boundary and its semantics are stable.

## Engineering Principle

The host snapshot is an integration boundary, not a dumping ground for subsystem logic.

Subsystems should continue to own:

* Linux interface knowledge
* parsing
* validation
* raw observations
* subsystem-specific calculations

The host layer should primarily own:

* composition
* observation boundaries
* partial collection state
* top-level host representation

Keeping that boundary clear allows future health, output, and monitoring layers to build on the collected observations without rewriting the underlying Linux interfaces.
