# Process CPU Utilization Experiment

This experiment measures CPU consumption for a Linux process using the process accounting counters exposed through `/proc/<pid>/stat`.

The goal is to understand how process CPU time is represented by the Linux kernel and how a point-in-time process CPU utilization measurement can be derived from cumulative counters.

## Purpose

The process collector already reads raw process accounting fields from `/proc/<pid>/stat`.

This experiment adds the next layer:

```text
/proc/<pid>/stat
        |
        v
  CPU counters
  utime + stime
        |
        v
   Sample 1
        |
      wait
        |
   Sample 2
        |
        v
    Counter delta
        |
        v
  CPU time consumed
        |
        v
Actual elapsed time
        |
        v
 Process utilization
````

The important idea is that Linux exposes cumulative CPU-time counters. Utilization is therefore calculated from the change between two observations rather than read directly from a single `/proc` value.

## Linux Source

The process CPU accounting data comes from:

```text
/proc/<pid>/stat
```

Relevant fields:

| Field | Name        | Meaning                                                    |
| ----- | ----------- | ---------------------------------------------------------- |
| 1     | `pid`       | Process ID                                                 |
| 14    | `utime`     | CPU time spent in user mode                                |
| 15    | `stime`     | CPU time spent in kernel mode                              |
| 22    | `starttime` | Time the process started after system boot, in clock ticks |

`utime` and `stime` are cumulative CPU-time counters measured in clock ticks.

The project currently uses:

```bash
getconf CLK_TCK
```

On the development system this reports:

```text
100
```

Therefore:

```text
100 ticks = 1 second of CPU time
```

The conversion is:

```text
CPU seconds = CPU ticks / CLK_TCK
```

## Sampling Model

A single process sample contains:

```go
type Sample struct {
    PID        int64
    StartTime  uint64
    UTime      uint64
    STime      uint64
    ObservedAt time.Time
}
```

The kernel counters remain in their native units.

`ObservedAt` records when the userspace observation was made.

Two samples are required:

```text
Sample 1:
    utime = U1
    stime = S1
    time  = T1

Sample 2:
    utime = U2
    stime = S2
    time  = T2
```

CPU time consumed during the interval is:

```text
(U2 - U1) + (S2 - S1)
```

The elapsed wall-clock time is:

```text
T2 - T1
```

CPU utilization is then:

```text
CPU time consumed
-----------------
 elapsed time
```

The implementation uses the actual timestamps recorded for the two observations rather than assuming that a requested one-second sleep produced exactly one second of elapsed time.

## Process Identity

A PID alone is not sufficient to identify a process over time.

Linux can reuse a PID after the original process exits.

For example:

```text
PID 5000
   |
   v
process A exits
   |
   v
PID 5000 becomes available
   |
   v
process B receives PID 5000
```

If a collector compared CPU counters using only:

```text
PID = 5000
```

it could incorrectly interpret process B's counters as a continuation of process A.

The collector therefore uses:

```text
(PID, StartTime)
```

as the process identity for CPU accounting.

If `StartTime` changes between samples, the samples are rejected as belonging to different process instances.

## Utilization Meaning

The reported utilization is relative to one CPU.

For example:

```text
1.00  = 100% of one CPU
0.50  = 50% of one CPU
0.25  = 25% of one CPU
```

This means a process using approximately one full CPU for the entire observation interval reports approximately:

```text
100%
```

A multithreaded process can consume more than one CPU simultaneously, so process utilization can exceed 100%.

The current implementation deliberately does not divide utilization by the number of CPUs in the host.

That distinction matters:

```text
process utilization
        |
        +-- single-CPU equivalent
        |
        +-- not host-wide CPU percentage
```

Host-wide CPU accounting is handled separately by the CPU collector.

## Controlled Experiment

A CPU-bound process was created with:

```bash
python3 - <<'PY' &
import time

end = time.monotonic() + 30

while time.monotonic() < end:
    pass
PY

pid=$!
echo "PID=$pid"
```

The process was then sampled using:

```bash
go run ./experiments/process-cpu "$pid"
```

Observed result:

```text
pid=69521
cpu_ticks=78
cpu_seconds=0.780
elapsed_seconds=1.002
utilization=77.84%
```

The process accumulated:

```text
78 CPU ticks
```

With:

```text
CLK_TCK = 100
```

that corresponds to:

```text
0.780 CPU seconds
```

The actual observation interval was:

```text
1.002 seconds
```

Therefore the measured utilization was approximately:

```text
0.780 / 1.002 = 77.84%
```

The result is below 100% even though the process was intentionally CPU-bound.

That is expected. A CPU-bound process requests CPU continuously, but the scheduler still determines when it actually runs. Other processes, kernel activity, virtualization overhead, and scheduling can reduce the CPU time received during the sampling interval.

## Process Lifetime Races

`/proc` is a live kernel interface.

The following sequence is possible:

```text
enumerate PID
    |
    v
process exits
    |
    v
read /proc/<pid>/stat
    |
    v
ENOENT
```

This is normal process churn.

The collector therefore treats a process disappearing between enumeration and stat collection as a normal race and skips it.

Other errors are not silently ignored.

Current policy:

| Condition             | Collector behavior |
| --------------------- | ------------------ |
| Process disappears    | Skip               |
| Zombie process        | Collect            |
| Permission denied     | Return error       |
| Malformed stat record | Return error       |
| Unexpected I/O error  | Return error       |

A zombie process can remain visible in `/proc/<pid>/stat` after the process has exited because its parent has not yet reaped it.

Therefore:

```text
running process
      |
      v
   exit()
      |
      v
   zombie
      |
      v
 wait()/waitpid()
      |
      v
/proc/<pid> disappears
```

The collector preserves the kernel-reported `Z` state rather than treating it as a missing process.

## Measurement Limitations

### `/proc` is not transactional

The process information is observed from a live kernel interface.

Different reads can occur at different points in the process lifecycle.

The collector does not claim to produce an atomic system-wide process snapshot.

### The sampling interval is approximate

The experiment requests a one-second delay, but the actual elapsed interval is measured independently.

For this reason:

```text
sleep duration != assumed measurement interval
```

The implementation uses:

```go
ObservedAt
```

from both samples and calculates the real elapsed duration.

### CPU counters are cumulative

`utime` and `stime` do not represent current CPU utilization.

They represent accumulated CPU time since the process started.

Utilization therefore requires at least two samples.

### Scheduler effects

A CPU-bound process does not guarantee exactly 100% utilization.

The process may be temporarily descheduled or compete with other runnable work.

### Virtualization

The experiment was performed inside a VirtualBox VM.

The measured CPU time therefore reflects the CPU time actually accounted to the process inside the guest environment. Host scheduling can influence what the guest process receives.

### Guest time

Linux reports guest CPU time through additional fields in `/proc/<pid>/stat`.

`utime` already includes guest time, so the current implementation does not add guest time separately.

Doing so would double-count CPU time.

### PID reuse

A PID can identify different process instances over time.

The implementation protects CPU deltas using `StartTime` in addition to PID.

## Implementation

The CPU sampling implementation is split into small pieces.

```text
internal/process/
├── sample.go
├── delta.go
├── utilization.go
├── delta_test.go
└── utilization_test.go
```

### `sample.go`

Converts raw process statistics into a timestamped CPU accounting sample.

### `delta.go`

Compares two samples and calculates:

* CPU tick delta
* CPU seconds
* elapsed wall-clock time
* process identity validity

It rejects:

* PID changes
* process identity changes
* counter rollback
* timestamps moving backwards
* non-positive elapsed intervals

### `utilization.go`

Converts the CPU-time delta into single-CPU-equivalent utilization.

### Tests

The implementation includes tests for:

* normal CPU deltas
* PID changes
* PID reuse
* CPU counter rollback
* invalid elapsed time
* full CPU utilization
* half CPU utilization
* partial CPU utilization

The test suite also continues to validate the existing process parser and collector.

## Current Boundary

This experiment establishes process CPU accounting.

It does not yet implement:

* process ranking
* top CPU processes
* process health policy
* process killing
* historical CPU storage
* a background daemon
* Prometheus metrics
* alerting
* Kubernetes integration
* container-specific accounting

Those belong to later layers.

The current boundary is deliberately small:

```text
Linux process accounting
        |
        v
Raw counters
        |
        v
Timestamped samples
        |
        v
Validated delta
        |
        v
CPU utilization
```