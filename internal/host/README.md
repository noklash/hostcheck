## Host Snapshot

Hostcheck now has a top-level host snapshot boundary that brings the individual subsystem collectors together.

The current collection path is:

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

The host snapshot currently contains observations for:

* CPU
* memory
* filesystem
* processes
* network

The snapshot records when collection began and preserves subsystem-level collection failures instead of discarding successful observations from other subsystems.

Collection is best-effort and non-atomic. The host is a live system, so different fields may represent slightly different points in time.

The host layer is deliberately concerned with collecting and composing observations. It does not currently apply health thresholds or decide whether an observation is healthy, warning, or critical. Those decisions belong to the health evaluation layer that will be introduced after the snapshot model is stable.

See [`docs/host-snapshot.md`](docs/host-snapshot.md) for the snapshot model, timestamp semantics, partial failures, live-system races, and current design boundaries.
