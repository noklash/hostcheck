# Process Collection Experiment

## Purpose

Observe process statistics collected from `/proc/<pid>/stat` through the
hostcheck process collector.

## What this experiment demonstrates

- `/proc` exposes one directory per visible process.
- Numeric directory names correspond to process IDs.
- Process statistics are read from `/proc/<pid>/stat`.
- The collector combines process enumeration, reading, and parsing.
- CPU accounting fields remain in kernel clock ticks.
- Process state, parent PID, thread count, and other raw fields can be observed
  from the collected snapshot.

## Run

```bash
go run ./experiments/process
````

## Example

```text
processes=253
pid=1 comm="systemd" state=S ppid=0 threads=1 utime=640 stime=555
pid=1012 comm="avahi-daemon" state=S ppid=1 threads=1 utime=11 stime=12
pid=1013 comm="chronyd-starter" state=S ppid=1 threads=1 utime=17 stime=12
...
```

The exact process count and accounting values vary between runs because `/proc`
represents live kernel state.

## Interpretation

`utime` and `stime` are cumulative CPU-time counters reported by Linux in clock
ticks. They are not CPU percentages and are not wall-clock seconds.

`state` is the process state reported by `/proc/<pid>/stat`.

`ppid` identifies the parent process.

`threads` reports the number of threads in the process.

The collector preserves these values in their kernel-facing representation.
Conversion into CPU seconds or utilization requires sampling the same process
at different times and is handled separately.

## Race Behavior

The `/proc` filesystem is live. A process may disappear after enumeration but
before `/proc/<pid>/stat` is read.

The collector treats `ENOENT` as expected process churn and skips that process.
Other read or parsing errors remain visible to the caller.

A process can also temporarily remain visible as a zombie after it exits and
before its parent reaps it. The collector preserves the reported `Z` state
rather than treating it as a collection failure.

## Limitations

This experiment performs a single process snapshot.

It does not calculate:

* per-process CPU utilization
* CPU percentage
* CPU time over an interval
* process lifetime
* process ranking
* health status

Those require additional sampling and policy layers.