# Health Evaluation

## Purpose

The health layer translates collected host observations into explicit operational assessments.

Collectors answer:

> What did Linux report?

Derived observation logic answers:

> What can we calculate from those observations?

Health evaluation answers:

> Given a defined policy, what does that observation mean operationally?

The health layer therefore sits after collection and derived observation logic. It should not reinterpret raw Linux data, hide uncertainty, or invent conclusions when the required observation is unavailable.

The current implementation supports independent health evaluation for:

* memory available capacity
* filesystem available block capacity
* filesystem available inode capacity
* CPU utilization

CPU utilization is evaluated separately from the single-snapshot health evaluator because CPU utilization requires two samples.

---

## Design Goals

The health layer is designed around a few principles.

### 1. Health policy is explicit

Thresholds are supplied by the caller rather than hidden inside collectors.

This keeps collection independent from operational policy.

For example:

```go
type MemoryPolicy struct {
    DegradedBelowPercent float64
    CriticalBelowPercent float64
}
```

The collector does not decide whether memory is healthy.

It only reports the observed memory state.

---

### 2. Assessment is separate from observation

A collector should not return:

```text
memory is critical
```

It should return measurements such as:

```text
MemAvailable = ...
MemTotal = ...
```

The health layer then derives available capacity and evaluates it against a policy.

This separation allows the same observations to support different operational policies.

---

### 3. Uncertainty must be represented explicitly

An observation can be unavailable or invalid without the host itself being unhealthy.

For example:

* `/proc/meminfo` cannot be read
* filesystem statistics cannot be collected
* inode totals are unavailable
* a required CPU sample is missing

These cases should not automatically become `critical`.

The health model therefore distinguishes between:

```text
assessable
unassessable
```

and:

```text
ok
degraded
critical
```

A health status only exists when the underlying observation is assessable.

---

### 4. Invalid policy is different from unhealthy state

A malformed policy is an evaluation error.

For example:

```text
critical threshold <= degraded threshold
```

is not evidence that the host is unhealthy.

It means the evaluator was given an invalid policy.

The evaluator should therefore return an error rather than produce a misleading assessment.

---

## Assessment Model

The health package uses a common assessment structure:

```go
type Assessment struct {
    Subject      string
    Availability Availability
    Status       Status
    Reason       string
    Evidence     []string
}
```

Availability describes whether the health rule had enough information to make an assessment.

```go
type Availability string

const (
    Assessable   Availability = "assessable"
    Unassessable Availability = "unassessable"
)
```

Status describes the operational state when the assessment is assessable.

```go
type Status string

const (
    OK       Status = "ok"
    Degraded Status = "degraded"
    Critical Status = "critical"
)
```

The important distinction is:

```text
unassessable != critical
```

A missing observation means the health rule cannot establish the state. It does not prove that the subsystem is unhealthy.

---

## Assessment Validation

Assessments are validated independently of the rules that produce them.

The validation rules include:

* subject must be present
* assessable assessments must contain a valid status
* unassessable assessments must not contain a status

This provides a consistent contract between individual health rules and higher-level consumers.

The health layer should fail clearly when an invalid assessment is constructed rather than allowing malformed health data to propagate.

---

# Memory Health

## Observation

The memory collector reads `/proc/meminfo` and produces a structured memory observation.

The health rule does not use `MemFree` as its primary capacity measurement.

Linux's `MemAvailable` estimate is used because it provides a more useful representation of memory that can be made available to applications without excessive swapping.

The derived observation is:

```text
available memory percentage
```

Conceptually:

```text
MemAvailable / MemTotal * 100
```

The calculation belongs to the memory-derived observation layer rather than the health evaluator.

The health evaluator consumes the resulting capacity value and applies policy.

---

## Policy

Memory health uses:

```go
type MemoryPolicy struct {
    DegradedBelowPercent float64
    CriticalBelowPercent float64
}
```

The policy defines lower capacity boundaries.

A healthy system has available memory above the degraded boundary.

Crossing the degraded boundary produces a degraded assessment.

Crossing the critical boundary produces a critical assessment.

The critical threshold must be strictly below the degraded threshold.

For example:

```text
degraded below 20%
critical below 10%
```

means:

```text
> 20%       -> OK
10-20%      -> Degraded
< 10%       -> Critical
```

The exact boundary behavior is defined by the evaluator tests and policy implementation.

---

## Evidence

The memory health rule includes the derived value as evidence.

Example:

```text
available_memory_percent=34.72
```

The evidence is intentionally machine-readable so that future JSON output or other consumers do not need to parse the human-readable reason.

---

# Filesystem Health

## Observation

Filesystem collection uses `statfs` rather than parsing the output of the `df` command.

The filesystem observation includes:

```go
type Stats struct {
    Path             string
    BlockSize        uint64
    BlocksTotal      uint64
    BlocksFree       uint64
    BlocksAvailable  uint64
    InodesTotal      uint64
    InodesFree       uint64
}
```

The health layer currently evaluates available filesystem block capacity separately from inode capacity.

The distinction matters because a filesystem can have:

* sufficient free blocks but exhausted inodes
* available inodes but insufficient storage capacity

These are different resource conditions and should therefore remain separate health subjects.

---

## Available Block Capacity

The derived filesystem observation calculates available block capacity using:

```text
BlocksAvailable / BlocksTotal * 100
```

`BlocksAvailable` is used instead of `BlocksFree` because it better represents space available to an unprivileged process.

The health evaluator then applies:

```go
type FilesystemPolicy struct {
    DegradedBelowPercent float64
    CriticalBelowPercent float64
}
```

The policy uses lower-capacity boundaries in the same general way as memory.

For example:

```text
degraded below 20%
critical below 10%
```

The health rule returns an assessment with:

```text
subject = filesystem
```

and evidence similar to:

```text
available_capacity_percent=37.42
```

---

# Filesystem Inode Health

Filesystem capacity and inode capacity are evaluated independently.

## Why Inodes Are Separate

A filesystem may still have significant byte capacity while having very few inodes remaining.

For example:

```text
disk capacity: 65% available
inode capacity: 3% available
```

The filesystem is not close to byte exhaustion, but it may be approaching inode exhaustion.

Treating these as one metric would hide an important failure mode.

---

## Derived Observation

The filesystem layer calculates available inode capacity from:

```text
InodesFree / InodesTotal * 100
```

The derived method returns an error when the total inode count is zero because a percentage cannot be meaningfully calculated.

This is treated as an inability to derive the required observation rather than as a critical filesystem condition.

---

## Policy

Filesystem inode health uses:

```go
type FilesystemInodePolicy struct {
    DegradedBelowPercent float64
    CriticalBelowPercent float64
}
```

The policy follows the same lower-capacity model as filesystem block capacity.

The critical threshold must be strictly below the degraded threshold.

The health subject is:

```text
filesystem_inodes
```

Example evidence:

```text
available_inode_percent=18.43
```

The evaluation flow is therefore:

```text
statfs
    |
    v
filesystem.Stats
    |
    v
AvailableInodePercent()
    |
    v
FilesystemInodePolicy
    |
    v
Assessment
```

This keeps inode calculation separate from the policy that determines whether the resulting value is acceptable.

---

# CPU Health

CPU health differs from memory and filesystem health because CPU utilization is not represented by a single cumulative observation.

## CPU Observation

Linux exposes cumulative CPU counters through `/proc/stat`.

The CPU observation contains counters such as:

```text
user
nice
system
idle
iowait
irq
softirq
steal
guest
guest_nice
```

These values represent accumulated CPU time.

A single sample cannot tell us CPU utilization over an interval.

Two samples are required:

```text
sample 1
   |
   | time passes
   v
sample 2
```

The difference between the samples provides the CPU time accumulated during the interval.

---

## CPU Delta

The CPU package calculates a delta between two cumulative CPU samples.

Counter regressions are rejected because cumulative CPU counters should not move backwards during normal observation.

A regression therefore indicates that the samples cannot safely be used to calculate utilization.

---

## CPU Utilization

CPU utilization is derived from the delta.

The calculation considers busy CPU time relative to total CPU time.

Guest and guest-nice time are not added again to the total because Linux already accounts for guest time within the user and nice counters.

This avoids double-counting CPU time.

The result is a percentage:

```text
0% -> completely idle
100% -> fully utilized
```

The derived utilization value is then passed to the health layer.

---

## CPU Policy

CPU health uses:

```go
type CPUUtilizationPolicy struct {
    DegradedAbovePercent float64
    CriticalAbovePercent float64
}
```

Unlike memory and filesystem capacity, CPU health uses upper boundaries because higher utilization represents increasing pressure.

The policy requires:

```text
0 <= degraded threshold <= 100
0 <= critical threshold <= 100
critical threshold > degraded threshold
```

The evaluator also rejects utilization values outside the valid range of:

```text
0-100%
```

---

## CPU Evaluation

CPU health is evaluated through a standalone evaluator:

```text
CPU sample 1
     |
     v
CPU sample 2
     |
     v
Delta()
     |
     v
Utilization()
     |
     v
CPUUtilizationPolicy
     |
     v
Assessment
```

The resulting assessment uses:

```text
subject = cpu
```

and includes evidence such as:

```text
utilization_percent=72.41
```

The CPU rule therefore follows the same general architecture as the other health rules while respecting the different sampling requirements of CPU accounting.

---

# Snapshot Evaluation

The host snapshot provides a boundary around observations collected from multiple Linux sources.

The current snapshot health evaluator evaluates the rules that can operate on the observations available within a single snapshot.

The policy is:

```go
type SnapshotPolicy struct {
    Memory          MemoryPolicy
    Filesystem      FilesystemPolicy
    FilesystemInode FilesystemInodePolicy
}
```

The snapshot evaluator currently produces assessments for:

```text
memory
filesystem
filesystem_inodes
```

CPU is intentionally excluded from this evaluator.

---

## Why CPU Is Not in Snapshot Evaluation

The `host.Snapshot` contains one cumulative CPU sample.

That is enough to preserve the observation but not enough to calculate CPU utilization.

CPU utilization requires:

```text
previous sample
+
current sample
+
elapsed observation interval
```

Adding CPU utilization directly to `EvaluateSnapshot()` would therefore blur the boundary between:

```text
single-snapshot evaluation
```

and:

```text
multi-sample evaluation
```

The current design keeps that distinction explicit.

The CPU rule can be evaluated independently once the caller has collected the required samples and derived utilization.

---

# Unassessable Conditions

Health evaluation should preserve uncertainty instead of manufacturing a status.

Examples include:

### Memory

If the memory observation is missing:

```text
memory -> unassessable
```

### Filesystem

If filesystem statistics cannot be collected:

```text
filesystem -> unassessable
```

### Filesystem Inodes

If inode capacity cannot be derived:

```text
filesystem_inodes -> unassessable
```

### CPU

If two valid CPU samples are not available:

```text
cpu -> cannot be evaluated
```

The exact representation depends on where the failure occurs.

The important rule is that missing data must not automatically become:

```text
critical
```

---

# Invalid Policies

Each evaluator validates its policy before evaluating the observation.

Examples of invalid policies include:

```text
degraded threshold outside valid range
critical threshold outside valid range
critical threshold not stricter than degraded threshold
```

The evaluator returns an error for an invalid policy.

This is intentionally different from returning:

```text
Assessment{
    Status: Critical,
}
```

A policy configuration error belongs to the caller or configuration layer.

It is not evidence about the host.

---

# Health Evaluation Flow

The overall architecture is:

```text
                 Linux
                   |
                   v
              Collectors
                   |
                   v
          Raw Observations
                   |
                   v
        Derived Observations
                   |
                   v
            Health Rules
                   |
                   v
             Assessments
```

For memory:

```text
/proc/meminfo
     |
     v
MemInfo
     |
     v
Available Capacity
     |
     v
MemoryPolicy
     |
     v
Assessment
```

For filesystem block capacity:

```text
statfs
  |
  v
Filesystem Stats
  |
  v
Available Capacity
  |
  v
FilesystemPolicy
  |
  v
Assessment
```

For filesystem inode capacity:

```text
statfs
  |
  v
Filesystem Stats
  |
  v
Available Inode Capacity
  |
  v
FilesystemInodePolicy
  |
  v
Assessment
```

For CPU utilization:

```text
/proc/stat sample 1
        |
        v
/proc/stat sample 2
        |
        v
      Delta
        |
        v
   Utilization
        |
        v
CPUUtilizationPolicy
        |
        v
    Assessment
```

---

# Current V1 Boundary

The current V1 health layer includes:

* explicit assessment availability
* explicit health status
* assessment subjects
* human-readable reasons
* machine-readable evidence
* assessment validation
* memory available-capacity health evaluation
* filesystem available block-capacity health evaluation
* filesystem available inode-capacity health evaluation
* snapshot-level evaluation for memory and filesystem rules
* CPU utilization health evaluation as a standalone rule
* explicit policy validation
* unassessable handling for missing or invalid observations
* separation between collection, derived observations, and health policy

The current design deliberately does not attempt to provide:

* a universal health score
* automatic weighting of subsystems
* automatic aggregation into a single host verdict
* historical trend analysis
* alert routing
* Prometheus integration
* Kubernetes health integration
* daemonized continuous monitoring
* automatic remediation
* CPU utilization inside the single-snapshot evaluator

These concerns can be considered later when there is a concrete requirement for them.

---

# Why There Is No Overall Host Verdict Yet

A host-level verdict such as:

```text
healthy
degraded
critical
```

looks simple but introduces policy questions that are not yet defined.

For example:

```text
memory = OK
filesystem = OK
filesystem_inodes = Degraded
cpu = Critical
```

What should the host status be?

A simple:

```text
worst status wins
```

rule would produce `critical`.

But that is still a policy decision.

A critical CPU utilization measurement over a short interval may have a very different operational meaning from an exhausted filesystem inode pool.

The health layer should therefore establish reliable subsystem assessments before introducing host-level aggregation.

The current architecture leaves room for aggregation later without forcing that decision into every individual rule.

---

# Testing Strategy

Health rules should be tested independently from Linux collection.

Tests should cover:

* valid policies
* invalid policies
* normal observations
* degraded observations
* critical observations
* invalid observations
* missing observations
* boundary conditions
* evidence formatting
* assessment validation

For CPU specifically, tests should also cover:

* valid utilization
* utilization outside `0-100%`
* valid threshold ordering
* invalid threshold ordering
* threshold boundaries
* generated evidence
* standalone evaluation semantics

This allows health policy behavior to be verified deterministically without depending on the current state of the host running the tests.

---

# Architectural Principle

The health layer should remain deliberately boring.

Collectors should understand Linux.

Derived observation logic should understand calculations.

Health rules should understand policy.

The host snapshot should provide the integration boundary.

The architecture should therefore remain:

```text
Linux semantics
      |
      v
Collection
      |
      v
Observation
      |
      v
Derived observation
      |
      v
Policy evaluation
      |
      v
Assessment
```

Each layer has a clear responsibility.

This is important for Hostcheck because the project is intended to become a reliable Linux infrastructure tool rather than a collection of loosely connected system checks.

---

# Current Engineering Direction

The first health rules establish the pattern for the rest of Hostcheck:

```text
collect
derive
validate
evaluate
explain
```

Memory establishes available-capacity evaluation.

Filesystem establishes block-capacity evaluation.

Filesystem inode capacity establishes that different resource dimensions should remain independently assessable.

CPU establishes that health evaluation must also account for observation semantics and sampling requirements.

The next health rules should follow the same discipline.

A new rule should only be introduced when:

1. the underlying observation is well defined
2. the derived value is understood
3. the operational policy is explicit
4. the assessment semantics are clear
5. the failure and uncertainty cases are defined
6. the rule can be tested independently

This keeps the health layer small, explainable, and suitable for later integration into the wider Hostcheck system.

---

# Summary

Hostcheck's health layer converts Linux observations into explicit operational assessments without mixing collection, calculation, and policy.

The current implemented rules cover:

```text
Memory
Filesystem block capacity
Filesystem inode capacity
CPU utilization
```

Memory, filesystem block capacity, and filesystem inode capacity can be evaluated from a single host snapshot.

CPU utilization requires multiple cumulative CPU samples, so it is currently evaluated through a standalone CPU health evaluator rather than being embedded in the single-snapshot evaluator.

This distinction keeps the system honest about what can and cannot be concluded from a single observation.

The core principle remains:

> Collect what Linux reports, derive what can be calculated, and only then apply explicit operational policy.
