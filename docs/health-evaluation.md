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

The current implementation supports health evaluation for:

* memory available capacity
* filesystem available block capacity
* filesystem available inode capacity
* CPU utilization
* process state, specifically processes observed in uninterruptible sleep (`D` state)
* network interface state
* network route state

Memory, filesystem, process-state, and network health can be evaluated from observations contained in a host snapshot when their corresponding policies are supplied.

CPU utilization remains a separate evaluation path because utilization requires multiple cumulative CPU samples rather than one snapshot.

The health package also provides explicit aggregation of assessments into a host-level `Result`. Aggregation is a separate policy boundary from the individual health rules.

---

# Design Goals

The health layer is designed around several principles.

## 1. Health policy is explicit

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

The same principle applies to filesystem capacity, inode capacity, CPU utilization, process state, network interfaces, and network routes.

---

## 2. Assessment is separate from observation

A collector should not return:

```text
memory is critical
```

It should return measurements such as:

```text
MemAvailable = ...
MemTotal = ...
```

The health layer then derives the required capacity value and evaluates it against a policy.

This separation allows the same observations to support different operational policies.

---

## 3. Uncertainty must be represented explicitly

An observation can be unavailable or invalid without the host itself being unhealthy.

Examples include:

* `/proc/meminfo` cannot be read
* filesystem statistics cannot be collected
* inode totals are unavailable
* a required CPU sample is missing
* process information cannot be collected
* network information cannot be collected

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

An unassessable assessment can still be included in a larger result so that consumers can see which health propositions could not be evaluated.

---

## 4. Invalid policy is different from unhealthy state

A malformed policy is an evaluation error.

For example:

```text
critical threshold <= degraded threshold
```

is not evidence that the host is unhealthy.

It means the evaluator was given an invalid policy.

The evaluator should therefore return an error rather than produce a misleading assessment.

Policy validation belongs to the health evaluation boundary and occurs before observations are interpreted.

---

## 5. Observation semantics must be respected

Different Linux observations have different meanings.

A cumulative CPU counter cannot be interpreted as utilization from one sample.

Likewise, observing a process in `D` state tells us that the process was in uninterruptible sleep at the time of collection. It does not establish how long it remained there or whether the condition persisted.

A network route observation tells us what route was reported by the kernel. It does not establish that the route is reachable from every destination or that an external service is available.

Health rules must therefore preserve the semantics and limitations of the underlying observations.

---

# Assessment Model

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

`Subject` identifies what is being evaluated.

`Availability` describes whether the health rule had enough information to make an assessment.

`Status` describes the operational state when the assessment is assessable.

`Reason` provides a human-readable explanation.

`Evidence` contains machine-readable supporting values.

---

## Availability

```go
type Availability string

const (
    Assessable   Availability = "assessable"
    Unassessable Availability = "unassessable"
)
```

An assessment is `assessable` when the required observation was available and the health rule could evaluate it.

An assessment is `unassessable` when the required observation was unavailable or could not be interpreted safely.

The important distinction is:

```text
unassessable != critical
```

A missing observation means the health rule cannot establish the state. It does not prove that the subsystem is unhealthy.

---

## Status

```go
type Status string

const (
    OK       Status = "ok"
    Degraded Status = "degraded"
    Critical Status = "critical"
)
```

A status is meaningful only for an assessable assessment.

An unassessable assessment does not receive an `OK`, `Degraded`, or `Critical` status.

This prevents missing data from being confused with an observed unhealthy condition.

---

# Assessment Validation

Assessments are validated independently of the rules that produce them.

The validation rules include:

* subject must be present
* assessable assessments must contain a valid status
* unassessable assessments must not contain a status

This provides a consistent contract between individual health rules and higher-level consumers.

The health layer should fail clearly when an invalid assessment is constructed rather than allowing malformed health data to propagate.

Validation also provides a boundary between individual evaluators and aggregation.

An aggregator should not need to understand how memory, filesystem, CPU, process, or network assessments were produced. It should be able to rely on the common assessment contract.

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

A healthy system has available memory at or above the configured degraded boundary.

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
>= 20%      -> OK
10-20%      -> Degraded
< 10%       -> Critical
```

The exact boundary behavior is defined by the evaluator implementation and tests.

---

## Evidence

The memory health rule includes the derived value as machine-readable evidence.

Example:

```text
available_percent=34.72
```

The evidence is intentionally kept separate from the human-readable reason so that future JSON output or other consumers do not need to parse explanatory text.

---

# Filesystem Health

## Observation

Filesystem collection uses `statfs` rather than parsing the output of the `df` command.

The filesystem observation includes:

```go
type Stats struct {
    Path            string
    BlockSize       uint64
    BlocksTotal     uint64
    BlocksFree      uint64
    BlocksAvailable uint64
    InodesTotal     uint64
    InodesFree      uint64
}
```

The health layer evaluates available filesystem block capacity separately from inode capacity.

The distinction matters because a filesystem can have:

* sufficient free blocks but exhausted inodes
* available inodes but insufficient storage capacity

These are different resource conditions and therefore remain separate health subjects.

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
available_percent=37.42
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

The evaluation flow is:

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

The evaluator also rejects utilization values outside:

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

CPU evaluation therefore follows the same general architecture as the other health rules while respecting the different sampling requirements of CPU accounting.

CPU utilization is not calculated from the single CPU observation stored in `host.Snapshot`.

---

# Process-State Health

Process-state health evaluates the number of processes observed in Linux's uninterruptible sleep state, represented by `D`.

A process in `D` state is waiting in uninterruptible sleep, commonly while waiting on a kernel resource such as I/O.

The presence of a `D`-state process is not by itself proof of host failure.

The health rule therefore evaluates the observed count against an explicit policy.

---

## Process-State Observation

The process collector obtains process state from process statistics exposed through `/proc`.

Each process contains a state value as part of its process statistics.

The health evaluator counts processes whose observed state is:

```text
D
```

The result is an observation of the number of processes in uninterruptible sleep at the collection point.

For example:

```text
d_state_processes=0
```

or:

```text
d_state_processes=3
```

The health rule does not infer persistence from this value.

A process observed in `D` state during one collection does not establish that it remained in that state for a particular duration.

---

## Process-State Policy

Process-state health uses:

```go
type ProcessStatePolicy struct {
    DegradedAtOrAbove int
    CriticalAtOrAbove int
}
```

The policy defines count thresholds.

For example:

```text
degraded at or above 2 processes
critical at or above 5 processes
```

means:

```text
0-1 D-state processes -> OK
2-4 D-state processes -> Degraded
5+ D-state processes  -> Critical
```

The critical threshold must be greater than the degraded threshold.

Negative thresholds are invalid.

The thresholds are intentionally supplied by the caller rather than embedded in the process collector.

---

## Process-State Evaluation

The evaluation flow is:

```text
/proc process observations
        |
        v
process.Process
        |
        v
count D-state processes
        |
        v
ProcessStatePolicy
        |
        v
Assessment
```

The resulting assessment uses:

```text
subject = process_state
```

and includes evidence such as:

```text
d_state_processes=3
```

A degraded assessment means that the observed D-state count reached the configured degraded threshold.

A critical assessment means that the observed count reached the configured critical threshold.

The rule does not claim that the processes are permanently stuck, that the underlying resource is definitely failing, or that the host is necessarily unhealthy overall.

---

## Process-State Evaluation in a Snapshot

Process-state health is now supported by `EvaluateSnapshot()` when a `ProcessStatePolicy` is explicitly supplied.

The snapshot policy contains:

```go
type SnapshotPolicy struct {
    Memory            MemoryPolicy
    Filesystem        FilesystemPolicy
    FilesystemInode   FilesystemInodePolicy
    ProcessState      *ProcessStatePolicy
    NetworkInterfaces []NetworkInterfacePolicy
    NetworkRoutes     []NetworkRoutePolicy
}
```

The pointer is intentional.

A `nil` `ProcessState` policy means that process-state health is not requested as part of the snapshot evaluation.

When a process-state policy is supplied, the evaluator uses the processes contained in the snapshot.

If process collection failed, the process-state assessment becomes unassessable rather than critical.

This preserves the distinction between:

```text
process observation unavailable
```

and:

```text
process observation available and D-state threshold exceeded
```

---

# Network Health

Network health evaluates selected network observations collected from the host.

The network collector obtains information about interfaces, addresses, and routes.

Network health currently operates on interface and route observations.

It does not perform active reachability probes.

This distinction is important because observing a local interface or route is different from proving that an external destination is reachable.

---

# Network Interface Health

Network interface health evaluates an explicitly selected interface against a caller-supplied policy.

The evaluator works from the network observation collected by Hostcheck rather than parsing the output of the `ip` command.

The policy identifies the interface and the conditions that should be considered acceptable.

This keeps interface policy separate from Linux interface collection.

The resulting assessment uses the subject:

```text
network_interface
```

The assessment includes evidence describing the observed interface state.

---

## Network Interface Semantics

An interface observation can establish facts such as:

* whether the interface exists in the collected observation
* the observed interface state
* addresses associated with the interface
* interface-level properties exposed by the collector

It does not automatically establish:

* Internet reachability
* application-layer connectivity
* DNS resolution
* remote service availability
* end-to-end network health

The health rule therefore evaluates only the semantics represented by the collected observation and configured policy.

---

# Network Route Health

Network route health evaluates an explicitly selected route against a caller-supplied policy.

Routes are collected through the Linux networking interface rather than by parsing human-readable command output.

The route observation contains information such as:

```text
destination
source
gateway
output interface
priority
table
protocol
scope
```

The route evaluator uses these structured observations to determine whether the requested route proposition can be assessed.

The resulting assessment uses the subject:

```text
network_route
```

---

## Route Semantics

A route observation describes what the Linux kernel reported in the routing table.

It does not establish that:

* the destination is reachable
* packets will successfully traverse every hop
* the gateway itself is reachable
* the remote service is available
* an application protocol will succeed

Active reachability testing is therefore outside the current network health boundary.

This keeps route health observational rather than turning it into an implicit probing system.

---

# Snapshot Evaluation

The host snapshot provides a boundary around observations collected from multiple Linux sources.

The snapshot health evaluator evaluates health rules that can operate on observations contained within a single snapshot.

The current policy is:

```go
type SnapshotPolicy struct {
    Memory            MemoryPolicy
    Filesystem        FilesystemPolicy
    FilesystemInode   FilesystemInodePolicy
    ProcessState      *ProcessStatePolicy
    NetworkInterfaces []NetworkInterfacePolicy
    NetworkRoutes     []NetworkRoutePolicy
}
```

The first three policy fields are always evaluated:

```text
memory
filesystem
filesystem_inodes
```

Process-state and network evaluations are enabled explicitly by supplying their corresponding policies.

This keeps the default snapshot evaluation focused while allowing callers to request additional health propositions when they have a policy for them.

---

## Snapshot Evaluation Flow

The snapshot evaluator follows this structure:

```text
host.Snapshot
      |
      v
EvaluateSnapshot()
      |
      +---- memory
      |
      +---- filesystem
      |
      +---- filesystem_inodes
      |
      +---- process state, if configured
      |
      +---- network interfaces, if configured
      |
      +---- network routes, if configured
      |
      v
[]Assessment
```

Each health rule remains responsible for evaluating its own observation.

`EvaluateSnapshot()` provides the integration boundary that connects those rules to the host snapshot.

It does not reinterpret the underlying observations.

---

# Why CPU Is Not in Snapshot Evaluation

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

Adding CPU utilization directly to `EvaluateSnapshot()` would blur the boundary between:

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

# Health Aggregation

Individual assessments answer questions about specific health propositions.

For example:

```text
memory = OK
filesystem = OK
filesystem_inodes = Degraded
```

Aggregation answers a different question:

> What overall health result can be established from these assessments?

The health package provides:

```go
type Result struct {
    Status      Status
    Coverage    Coverage
    Assessments []Assessment
}
```

Aggregation is performed explicitly through the health aggregation layer.

---

## Coverage

Coverage describes how completely the supplied health rules were evaluated.

```go
type Coverage string

const (
    Complete   Coverage = "complete"
    Partial    Coverage = "partial"
    Unavailable Coverage = "unavailable"
)
```

The meanings are:

```text
complete
```

All supplied assessments were assessable.

```text
partial
```

At least one assessment was assessable, but at least one supplied assessment was unassessable.

```text
unavailable
```

No assessment was assessable, or there were no assessments to evaluate.

Coverage is important because a status without knowing how much of the system was actually evaluated can be misleading.

For example:

```text
status = critical
coverage = partial
```

means the system established a critical condition among the assessments that were available, but some requested health propositions could not be evaluated.

---

# Aggregation Semantics

The current aggregation policy uses severity ordering:

```text
OK
Degraded
Critical
```

Among assessable assessments:

```text
Critical > Degraded > OK
```

Therefore:

```text
OK + OK
```

produces:

```text
OK
```

while:

```text
OK + Degraded
```

produces:

```text
Degraded
```

and:

```text
Degraded + Critical
```

produces:

```text
Critical
```

An unassessable assessment does not directly contribute a severity status.

Instead, it affects coverage.

For example:

```text
memory             = OK
filesystem         = OK
filesystem_inodes  = unassessable
```

produces an assessable host result with:

```text
status   = OK
coverage = Partial
```

This means the available evidence is healthy, but the evaluator cannot claim complete coverage.

---

# Why Aggregation Is Explicit

Aggregation is itself a policy decision.

A host-level result cannot simply be treated as another raw observation.

For example:

```text
memory = OK
filesystem = OK
filesystem_inodes = Degraded
```

could reasonably produce:

```text
Degraded
```

under a severity-based aggregation model.

However, more sophisticated systems might eventually consider:

* subsystem importance
* persistence
* observation confidence
* historical state
* service ownership
* dependency relationships
* remediation state
* workload context

Those concerns are outside the current V1 aggregation model.

The current implementation deliberately uses a simple and deterministic severity ordering while preserving the individual assessments and coverage information.

This allows a future aggregation policy to evolve without destroying the underlying subsystem evidence.

---

# `health.Evaluate()`

The health package provides a higher-level evaluation boundary:

```go
func Evaluate(
    snapshot host.Snapshot,
    policy SnapshotPolicy,
) (Result, error)
```

The function combines snapshot assessment and aggregation:

```text
host.Snapshot
      |
      v
EvaluateSnapshot()
      |
      v
[]Assessment
      |
      v
Aggregate()
      |
      v
health.Result
```

This provides the application layer with one explicit operation for evaluating a collected host snapshot.

The application does not need to manually call each individual health rule.

At the same time, the lower-level evaluators remain available for cases where a caller needs a specific health proposition, such as CPU utilization or a standalone process-state evaluation.

---

# CPU and Aggregation

CPU utilization remains outside `EvaluateSnapshot()` because it requires multiple samples.

Therefore CPU does not automatically appear in the `health.Result` returned by:

```go
health.Evaluate(snapshot, policy)
```

unless a separate application-level flow evaluates CPU and combines that assessment deliberately.

This distinction is intentional.

The snapshot evaluator operates on observations contained in one snapshot.

CPU utilization is derived from multiple snapshots.

A future application layer may choose to evaluate CPU separately and aggregate the resulting assessment with snapshot-based assessments, but that must remain an explicit orchestration decision rather than an accidental property of the CPU collector.

---

# Process-State and Aggregation

Process-state health can now participate in snapshot evaluation when a `ProcessStatePolicy` is supplied.

Therefore a configured snapshot can produce:

```text
memory
filesystem
filesystem_inodes
process_state
```

assessments.

The process-state rule still does not claim persistence or diagnosis.

For example:

```text
process_state = degraded
```

means:

> The observed number of processes in `D` state reached the configured degraded threshold during this collection.

It does not mean:

> The host is definitely experiencing an I/O failure.

Aggregation may incorporate the resulting assessment because the caller explicitly requested process-state health, but the individual assessment remains the source of truth about what was actually observed.

---

# Network Health and Aggregation

Network interface and route assessments can also participate in snapshot evaluation when their corresponding policies are supplied.

For example:

```text
network_interface
network_route
```

may appear alongside memory and filesystem assessments.

The aggregation layer treats them according to the same assessment contract as every other health rule.

This means the aggregator does not need to know whether an assessment originated from:

* memory
* filesystem
* inode capacity
* process state
* network interface
* network route

It only evaluates:

```text
Availability
Status
```

while preserving:

```text
Subject
Reason
Evidence
```

for consumers.

---

# Unassessable Conditions

Health evaluation should preserve uncertainty instead of manufacturing a status.

## Memory

If the memory observation is missing:

```text
memory -> unassessable
```

If memory collection failed, the reason can identify the collection failure.

---

## Filesystem

If filesystem statistics cannot be collected:

```text
filesystem -> unassessable
```

---

## Filesystem Inodes

If inode capacity cannot be derived:

```text
filesystem_inodes -> unassessable
```

For example, zero total inodes makes the percentage undefined.

This is an observation derivation failure, not evidence that the filesystem is critically unhealthy.

---

## Process State

If process collection fails and process-state health was requested:

```text
process_state -> unassessable
```

The evaluator must not manufacture a critical process-state result.

---

## Network

If network collection fails and network health was requested:

```text
network_interface -> unassessable
network_route     -> unassessable
```

The failure is represented as unavailable network observation rather than as a critical network condition.

---

## CPU

If two valid CPU samples are not available, CPU utilization cannot be derived.

The CPU evaluator therefore cannot establish a health status from insufficient samples.

The important distinction remains:

```text
observation unavailable
```

versus:

```text
observation available and policy threshold exceeded
```

---

# Invalid Policies

Each evaluator validates its policy before evaluating the observation.

Examples include:

```text
degraded threshold outside valid range
critical threshold outside valid range
critical threshold not stricter than degraded threshold
negative process-state threshold
critical process-state threshold not greater than degraded threshold
invalid network policy
```

The evaluator returns an error for an invalid policy.

This is intentionally different from returning:

```go
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
                   |
                   v
              Aggregation
                   |
                   v
             health.Result
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

For process-state health:

```text
/proc process observations
        |
        v
   Process states
        |
        v
 Count D-state processes
        |
        v
ProcessStatePolicy
        |
        v
    Assessment
```

For network interface health:

```text
Linux network state
        |
        v
Network observation
        |
        v
Interface policy
        |
        v
Assessment
```

For network route health:

```text
Linux routing state
        |
        v
Route observation
        |
        v
Route policy
        |
        v
Assessment
```

For snapshot evaluation:

```text
host.Snapshot
      |
      v
EvaluateSnapshot()
      |
      +---- memory
      +---- filesystem
      +---- filesystem_inodes
      +---- process state, if configured
      +---- network interfaces, if configured
      +---- network routes, if configured
      |
      v
[]Assessment
```

For complete health evaluation:

```text
host.Snapshot
      |
      v
health.Evaluate()
      |
      +---- EvaluateSnapshot()
      |
      v
[]Assessment
      |
      v
Aggregate()
      |
      v
health.Result
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
* CPU utilization health evaluation as a standalone multi-sample rule
* process-state health evaluation
* network interface health evaluation
* network route health evaluation
* snapshot-level evaluation
* optional process-state evaluation within snapshot evaluation
* optional network interface evaluation within snapshot evaluation
* optional network route evaluation within snapshot evaluation
* explicit policy validation
* unassessable handling for missing or invalid observations
* separation between collection, derived observations, and health policy
* explicit handling of CPU multi-sample semantics
* explicit handling of process-state snapshot semantics
* explicit handling of network observation semantics
* deterministic aggregation of assessable health statuses
* explicit health coverage
* preservation of individual assessments inside the final result
* a higher-level `health.Evaluate()` boundary

The current design deliberately does not attempt to provide:

* a universal health score
* automatic weighting of subsystems
* historical trend analysis
* persistent D-state detection
* active network reachability probing
* alert routing
* Prometheus integration
* Kubernetes health integration
* daemonized continuous monitoring
* automatic remediation
* CPU utilization directly inside the single-snapshot evaluator
* automatic diagnosis of the underlying cause of a health condition

These concerns can be considered later when there is a concrete requirement for them.

---

# Why CPU Remains Outside Snapshot Evaluation

The distinction between snapshot health and CPU health remains important.

A snapshot contains:

```text
CPU sample
```

while CPU utilization requires:

```text
CPU sample 1
+
CPU sample 2
+
elapsed interval
```

The snapshot therefore preserves the CPU observation without pretending that it already represents utilization.

The application layer can later orchestrate:

```text
previous snapshot
        |
        v
current snapshot
        |
        v
CPU delta
        |
        v
CPU utilization
        |
        v
CPU assessment
```

This keeps sampling semantics explicit.

---

# Why Aggregation Is Not Diagnosis

Aggregation answers:

> What status follows from the configured assessments?

It does not answer:

> Why is the host in that state?

For example:

```text
filesystem_inodes = Critical
```

does not identify which application consumed the inodes.

Likewise:

```text
network_route = Degraded
```

does not prove why the route is unsuitable.

And:

```text
process_state = Critical
```

does not establish the underlying storage or kernel condition.

The individual assessment's reason and evidence explain what was observed and how the policy interpreted it.

Diagnosis is a separate problem.

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

Snapshot evaluation should additionally test:

* healthy snapshot
* degraded snapshot
* critical snapshot
* missing memory observation
* missing filesystem observation
* invalid snapshot policy
* process-state policy enabled
* process collection failure
* network policy enabled
* network collection failure
* multiple assessments returned from one snapshot

Aggregation should test:

* all assessments `OK`
* degraded dominating `OK`
* critical dominating degraded
* partial coverage
* all assessments unassessable
* empty assessment set
* invalid assessment input
* result validation

For CPU specifically, tests should cover:

* valid utilization
* utilization outside `0-100%`
* valid threshold ordering
* invalid threshold ordering
* threshold boundaries
* generated evidence
* standalone evaluation semantics
* counter regression handling

For process-state health, tests should cover:

* valid process-state policies
* negative threshold rejection
* invalid threshold ordering
* zero D-state processes
* D-state count below the degraded threshold
* D-state count at the degraded threshold
* D-state count below the critical threshold
* D-state count at the critical threshold
* generated evidence
* standalone evaluation semantics

For network health, tests should cover:

* valid interface policies
* invalid interface policies
* interface present
* interface missing
* valid route policies
* invalid route policies
* route present
* route missing
* generated evidence
* unassessable network observations

This allows health policy behavior to be verified deterministically without depending on the current state of the host running the tests.

---

# Architectural Principle

The health layer should remain deliberately boring.

Collectors should understand Linux.

Derived observation logic should understand calculations.

Health rules should understand policy.

The host snapshot should provide the integration boundary.

Aggregation should combine validated assessments without pretending to diagnose their underlying causes.

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
      |
      v
Aggregation
      |
      v
Result
```

Each layer has a clear responsibility.

This is important for Hostcheck because the project is intended to become a reliable Linux infrastructure tool rather than a collection of loosely connected system checks.

---

# Current Engineering Direction

The health rules establish a common pattern:

```text
collect
derive
validate
evaluate
explain
aggregate
```

Memory establishes available-capacity evaluation.

Filesystem establishes block-capacity evaluation.

Filesystem inode capacity establishes that different resource dimensions should remain independently assessable.

CPU establishes that health evaluation must account for observation semantics and sampling requirements.

Process-state health establishes that process observations can support explicit operational policy while still requiring restraint about what a single observation can prove.

Network interface and route health establish that structured kernel networking observations can support explicit local network policy without automatically turning observation into active reachability testing.

Snapshot evaluation establishes the boundary for combining multiple single-observation health rules.

Aggregation establishes the boundary for producing a deterministic host-level result while preserving coverage and the underlying assessments.

A new health rule should only be introduced when:

1. the underlying observation is well defined
2. the derived value is understood
3. the operational policy is explicit
4. the assessment semantics are clear
5. the failure and uncertainty cases are defined
6. the rule can be tested independently

A new aggregation rule should likewise only be introduced when its operational meaning is explicitly defined.

This keeps the health layer small, explainable, and suitable for later integration into the wider Hostcheck system.

---

# Application Integration Boundary

The health package now exposes a clean boundary for the application layer.

The intended one-shot V1 flow is:

```text
cmd/hostcheck
      |
      v
application policy
      |
      v
host.Collect()
      |
      v
host.Snapshot
      |
      v
health.Evaluate()
      |
      v
health.Result
      |
      v
output
```

The application chooses the policy.

The host package collects observations.

The health package evaluates those observations.

The aggregation layer combines the resulting assessments.

The output layer presents the result.

No collector should contain operational thresholds.

No health evaluator should parse command output.

No output formatter should determine health status.

No application-level code should need to reproduce health evaluation logic.

This provides a clean boundary for the next stage of Hostcheck development.

---

# What the Health Result Represents

A final `health.Result` should be interpreted as a statement about the evidence that was evaluated under a particular policy.

It is not a permanent description of the host.

It represents:

```text
observations
+
derived values
+
configured policy
+
assessment coverage
```

at the time of evaluation.

For example:

```text
status   = degraded
coverage = complete
```

means that all requested assessments were available and the aggregate policy determined that at least one assessable condition was degraded.

Where:

```text
status   = ok
coverage = partial
```

the result means that all available assessments were healthy, but at least one requested assessment could not be established.

The distinction is important for infrastructure tooling because an absence of evidence should not silently become evidence of health.

---

# Summary

Hostcheck's health layer converts Linux observations into explicit operational assessments without mixing collection, calculation, and policy.

The current implemented rules cover:

```text
Memory
Filesystem block capacity
Filesystem inode capacity
CPU utilization
Process state
Network interface state
Network route state
```

Memory, filesystem block capacity, filesystem inode capacity, process state, and configured network rules can be evaluated from a host snapshot.

CPU utilization requires multiple cumulative CPU samples, so it remains a standalone multi-sample health evaluation rather than being embedded in the single-snapshot evaluator.

Snapshot evaluation provides the integration boundary for combining health rules that can operate on observations contained within one snapshot.

The resulting assessments can then be passed through the aggregation layer to produce:

```text
health.Result
```

which contains:

```text
Status
Coverage
Assessments
```

The aggregate status provides a deterministic severity summary.

The coverage value preserves the distinction between complete and partial evidence.

The individual assessments preserve the detailed operational evidence and reasoning.

Process-state health remains deliberately conservative. A D-state count describes what was observed at collection time and does not establish persistence or identify the underlying cause.

Network health remains observational. Interface and route observations do not automatically establish end-to-end reachability.

CPU health remains sampling-aware. A single cumulative CPU sample cannot be mistaken for utilization.

The core principle remains:

> Collect what Linux reports, derive what can be calculated, apply explicit operational policy, preserve uncertainty, and only then aggregate the resulting assessments.
