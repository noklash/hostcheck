# CPU Accounting Experiment

## Purpose

Verify that the CPU accounting model implemented by hostcheck works against live Linux `/proc/stat` data.

The experiment reads the aggregate CPU accounting counters twice, separated by approximately one second, then calculates CPU utilization from the counter delta.

## Linux source

The experiment reads:

    /proc/stat

The aggregate CPU record has the form:

    cpu user nice system idle iowait irq softirq steal guest guest_nice

The counters are cumulative CPU accounting values measured in clock ticks.

## System used

Observed development environment:

- Logical CPUs: 2
- `CLK_TCK`: 100

With two logical CPUs and `CLK_TCK=100`, aggregate CPU accounting can advance at approximately 200 ticks per second of wall-clock time.

## Method

The experiment:

1. Reads the aggregate `cpu` record from `/proc/stat`.
2. Waits approximately one second.
3. Reads the aggregate `cpu` record again.
4. Parses both samples with `cpu.ParseStat`.
5. Calculates the counter delta with `Stat.Delta`.
6. Calculates utilization with `cpu.Utilization`.

CPU utilization is calculated from counter deltas:

    total =
        user + nice + system + idle +
        iowait + irq + softirq + steal

    busy =
        user + nice + system +
        irq + softirq + steal

    utilization = busy / total * 100

`guest` and `guest_nice` are retained in the raw accounting model but are not independently added to the total because doing so would risk double-counting CPU time.

## Observed result

One real run of the Go experiment produced:

    CPU utilization: 22.75%

This is an observation from one sampling interval and should not be interpreted as a fixed property of the host.

## Important limitation

The experiment uses `time.Sleep` to separate the samples. The actual interval is not guaranteed to be exactly one second.

CPU utilization is therefore derived from the CPU accounting counters themselves rather than assuming a fixed number of ticks per second.

The experiment is also a short bounded observation of live kernel state. It is not a transactional snapshot of the entire system.

## What this proves

The experiment verifies the composition of the current CPU implementation:

    /proc/stat
        -> ParseStat
        -> Stat
        -> Delta
        -> Utilization

It demonstrates that the parser, counter-delta calculation, and utilization calculation can operate together against real Linux kernel data.

It does not yet implement the production hostcheck CPU collector or define CPU health thresholds.