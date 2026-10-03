# Health Evaluation

## Purpose

Hostcheck collects observations about a Linux host.

Those observations describe what the operating system exposed at collection time. They do not, by themselves, constitute a health verdict.

Health evaluation is the layer that determines what Hostcheck can reasonably conclude from the observations available in a host snapshot.

The health layer must preserve the distinction between:

* what was observed
* what can be assessed from those observations
* what interpretation is justified
* what health policy says about that interpretation

The goal of V1 is to establish a defensible health evaluation boundary without introducing arbitrary thresholds or pretending that unavailable evidence represents a known healthy or unhealthy state.

The current implementation provides independent subsystem assessments for memory and filesystem capacity. These assessments are evaluated from a `host.Snapshot` using explicit caller-supplied policies.

An overall host health verdict is deliberately not implemented yet.

---

## Observation vs Assessment

Hostcheck currently collects subsystem observations directly from Linux interfaces such as:

* `/proc`
* `/sys`
* `statfs`
* rtnetlink

The subsystem packages are responsible for obtaining and interpreting those operating-system interfaces.

Examples include:

* CPU accounting counters from `/proc/stat`
* memory values from `/proc/meminfo`
* filesystem capacity and inode statistics from `statfs`
* process information from `/proc/<pid>`
* network interface information from `/sys/class/net`
* network addresses and routes through Linux networking interfaces

These are observations.

Some observations can then be transformed into derived observations.

For example:

```text
/proc/stat CPU counters
        ↓
counter delta
        ↓
CPU utilization
```

The resulting utilization is still an observation derived from Linux accounting data. It is not automatically a health verdict.

Health evaluation occurs after this observation stage:

```text
Linux
  ↓
Collection
  ↓
Raw observations
  ↓
Subsystem-derived observations
  ↓
Health evaluation
  ↓
Assessment
```

This separation prevents subsystem collectors from embedding health policy into Linux parsing or accounting code.

---

## What Health Evaluation Answers

Health evaluation should answer:

> What can Hostcheck conclude about the host from the observations available in this snapshot?

It should not answer:

> What should the operator do?

Operational actions such as restarting a service, terminating a process, changing a route, expanding a filesystem, or replacing hardware are outside the health evaluation boundary.

Hostcheck should first establish what the system currently shows.

Policy and remediation can build on that information later.

---

## Assessability

An observation being present does not necessarily mean that it is sufficient for health assessment.

Health evaluation therefore distinguishes between observations that are available and observations that provide enough evidence for a particular assessment.

Consider CPU utilization.

A single CPU counter sample can be collected successfully:

```text
CPU counters
```

But utilization is based on change over time:

```text
sample 1
   ↓
delta
   ↓
sample 2
```

A single snapshot therefore does not necessarily contain enough evidence to calculate CPU utilization.

This gives the health pipeline an important distinction:

```text
Observed
   ↓
Assessable
   ↓
Interpretable
   ↓
Assessment
```

Not every observation must pass through every stage.

Some observations are useful for reporting without being sufficient for a health conclusion.

The health layer must not manufacture missing evidence.

---

## Collection Success Is Not Health

A successful collection does not mean that a subsystem is healthy.

For example:

```text
Memory collection succeeded
```

means that Hostcheck successfully read and interpreted the relevant memory information.

It does not mean:

```text
Memory is healthy
```

Likewise:

```text
Network collection succeeded
```

means that Hostcheck obtained the configured network observations.

It does not establish:

```text
Network connectivity is healthy
```

Collection and health are separate concerns.

The reverse is also important.

A collection failure does not automatically mean that the subsystem itself is unhealthy.

For example:

```text
CPU        collected
Memory     collected
Filesystem collected
Process    collected
Network    collection failed
```

The correct conclusion is that network health could not be assessed from this snapshot.

It is not valid to automatically convert the collection failure into:

```text
Network = CRITICAL
```

The failure describes Hostcheck's ability to observe the subsystem, not necessarily the subsystem's actual condition.

---

## Collection Errors

The host snapshot preserves collection errors:

```go
type CollectionError struct {
    Subsystem string
    Err       error
}
```

A snapshot can therefore contain both successful observations and collection failures.

For example:

```text
CPU        available
Memory     available
Filesystem available
Processes  available
Network    unavailable
```

The health layer must retain that distinction.

A failed collector should result in an assessment that indicates insufficient evidence rather than an invented health state.

This allows Hostcheck to say:

```text
Network could not be assessed because collection failed.
```

instead of incorrectly claiming:

```text
Network is unhealthy.
```

Collection errors remain evidence about the snapshot and are not themselves health statuses.

---

## Assessment Model

The current V1 health model separates two concepts.

### Availability

Availability describes whether sufficient evidence exists to perform a particular assessment.

Possible states are:

```text
Assessable
Unassessable
```

### Status

Status describes the health conclusion when sufficient evidence exists.

Possible states are:

```text
OK
DEGRADED
CRITICAL
```

These concepts are represented explicitly by the health assessment model.

Conceptually:

```go
type Assessment struct {
    Subject      string
    Availability Availability
    Status       Status
    Reason       string
    Evidence     []string
}
```

An assessable assessment must have a valid health status.

An unassessable assessment must not contain a health status.

For example:

```text
Subject: memory
Availability: unassessable
Status:
Reason: memory observation is unavailable
```

is different from:

```text
Subject: memory
Availability: assessable
Status: degraded
Reason: available memory capacity is below the degraded policy threshold
```

The first describes an evidence problem.

The second describes a health condition.

The distinction is important because treating both as `UNKNOWN` would hide why the assessment could not or did conclude something.

---

## Assessment Validation

Health assessments have explicit validation invariants.

An assessment must:

* identify a subject
* use a valid availability state
* provide a valid status when assessable
* omit status when unassessable

Conceptually:

```text
Assessable
    ↓
requires OK / DEGRADED / CRITICAL

Unassessable
    ↓
must not contain a status
```

This validation prevents malformed health results from propagating into later output or aggregation layers.

The health layer therefore validates both the semantic state and the relationship between availability and status.

---

## Evidence

A health assessment should be explainable from the observations that produced it.

The current assessment model retains:

```text
Assessment

├── subject
├── availability
├── status
├── reason
└── evidence
```

The evidence belongs to the assessment because it allows the result to be understood and later inspected.

For example, a filesystem assessment can conceptually contain:

```text
Subject: filesystem

Availability: assessable

Status: degraded

Reason:
available filesystem capacity is below the degraded policy threshold

Evidence:
available_percent=19.50
```

The health layer should not replace the underlying observation with the status.

The observation remains the evidence.

The assessment is the interpretation of that evidence under an explicit policy.

---

## Health Policy vs Observation

Hostcheck must keep operating-system observations separate from health policy.

For example, filesystem collection can produce:

```text
AvailableBytes
AvailablePercent
UsedBytes
UsedInodes
AvailableInodePercent
```

Those values describe the filesystem.

They do not inherently define:

```text
OK
DEGRADED
CRITICAL
```

A rule such as:

```text
available space below X percent = DEGRADED
```

is a policy decision.

The collector should therefore not contain arbitrary health thresholds.

The same principle applies to memory, CPU, processes, and networking.

The system should first establish what Linux reports.

Health policy can then decide what those observations mean operationally.

---

## Explicit Health Policy

Health policies are supplied to the health evaluation layer rather than embedded inside Linux collectors.

The current implementation defines separate policies for memory and filesystem capacity.

Conceptually:

```go
type MemoryPolicy struct {
    DegradedBelowPercent float64
    CriticalBelowPercent float64
}

type FilesystemPolicy struct {
    DegradedBelowPercent float64
    CriticalBelowPercent float64
}
```

The policy boundaries follow these semantics:

```text
available >= degraded threshold
    → OK

critical threshold <= available < degraded threshold
    → DEGRADED

available < critical threshold
    → CRITICAL
```

The exact threshold values are supplied by the caller.

Hostcheck therefore does not currently establish universal default thresholds such as:

```text
Memory < 10% = CRITICAL
Filesystem < 10% = CRITICAL
```

Those values remain policy.

---

## Policy Validation

Health policy must itself be valid before an assessment can be produced.

The current policy rules require:

* degraded threshold between 0 and 100
* critical threshold between 0 and 100
* critical threshold strictly below degraded threshold

For example:

```text
Degraded: 20%
Critical: 10%
```

is valid.

While:

```text
Degraded: 10%
Critical: 20%
```

is invalid.

Invalid policy is different from insufficient observation evidence.

Therefore:

```text
invalid policy
    ↓
evaluation error
```

rather than:

```text
invalid policy
    ↓
Unassessable
```

This distinction is important.

An unassessable result describes a limitation in the evidence.

An evaluation error describes a problem with the policy supplied to the evaluator.

---

## Current Health Evaluation Boundary

The current implementation establishes the following boundary:

```text
Host Snapshot
      ↓
Validate health policy
      ↓
Evaluate subsystem observations
      ↓
Produce independent subsystem assessments
```

The current snapshot evaluator evaluates:

* memory available capacity
* filesystem available capacity

The evaluator returns a collection of independent `Assessment` values.

It does not currently produce:

```text
Overall Host = OK
```

or:

```text
Overall Host = DEGRADED
```

That aggregation decision remains deliberately unresolved.

---

## Memory

Hostcheck collects Linux memory information from `/proc/meminfo`.

Memory observations include values such as:

* total memory
* available memory
* free memory
* buffers
* cached memory
* swap information where exposed

The health layer uses the semantics of these Linux fields rather than treating similarly named values as interchangeable.

In particular, `MemFree` must not automatically be interpreted as the amount of memory available to applications.

`MemAvailable` provides a more meaningful basis for understanding memory availability on modern Linux systems.

The memory package derives available capacity as:

```text
AvailablePercent =
    Available / Total × 100
```

The derivation is performed by the memory observation layer.

Health evaluation then applies an explicit `MemoryPolicy` to that derived observation.

The current memory health flow is therefore:

```text
/proc/meminfo
      ↓
MemInfo
      ↓
AvailablePercent
      ↓
MemoryPolicy
      ↓
Assessment
```

For example:

```text
AvailablePercent = 50%
Policy:
    degraded below 20%
    critical below 10%

Result:
    OK
```

If the memory observation is missing:

```text
Memory observation unavailable
```

the result is:

```text
Availability: Unassessable
Status: empty
```

If the observation itself is invalid, such as:

```text
Available > Total
```

the memory assessment is also unassessable.

The health layer does not reinterpret invalid memory accounting as a health failure.

---

## Filesystem

Filesystem observations include:

* total blocks
* free blocks
* available blocks
* used blocks
* available bytes
* used bytes
* available percentage
* used inodes
* available inode percentage

These are legitimate health inputs because they describe finite host resources that can become constrained.

The filesystem observation layer already derives:

```text
AvailablePercent
```

from the filesystem block statistics.

Health evaluation then applies an explicit `FilesystemPolicy`.

The current filesystem health flow is:

```text
statfs
  ↓
filesystem.Stats
  ↓
AvailablePercent
  ↓
FilesystemPolicy
  ↓
Assessment
```

For example:

```text
AvailablePercent = 8%

Policy:
    degraded below 20%
    critical below 10%
```

produces:

```text
Status: CRITICAL
```

because the observed value is below the configured critical boundary.

The meaning of the value still comes from policy.

The observation itself does not inherently mean `CRITICAL`.

The same applies to inode availability.

A filesystem may have available storage capacity while being constrained by inode exhaustion, or the reverse.

Health evaluation should therefore avoid reducing filesystem health to a single capacity percentage.

The current concrete filesystem rule evaluates available block capacity only.

Inode health remains a separate proposition that can be introduced when its policy and semantics are justified.

---

## CPU

Available observations include CPU accounting counters and derived utilization where sufficient samples exist.

CPU counters themselves are not a health verdict.

A future CPU health rule may use derived utilization or other explicit evidence, but the rule must define:

* required observations
* required sampling relationship
* calculation semantics
* policy boundaries

A single CPU snapshot should not be treated as sufficient evidence for utilization-based health.

The current health layer does not produce a CPU health assessment.

---

## Processes

Hostcheck collects process-level observations including process identity, state, CPU information, memory information, and other `/proc/<pid>` data where available.

Process observations are currently primarily informational.

The existence of a process, its process state, or its CPU consumption does not automatically establish a host health condition.

For example:

```text
process exists
```

does not imply:

```text
host healthy
```

and:

```text
process uses high CPU
```

does not automatically imply:

```text
host unhealthy
```

Process health becomes meaningful when a specific policy establishes what the process represents.

A future configuration could explicitly define requirements such as:

```text
required process X must exist
```

or:

```text
process Y must remain below a defined resource boundary
```

Such rules belong to health policy, not generic process collection.

The current health layer does not produce a process health assessment.

---

## Network

Hostcheck currently observes several distinct parts of Linux networking:

* interface identity
* interface state
* interface statistics
* addresses
* address scope
* routes
* route metadata
* multipath next hops where exposed

These observations must not be collapsed into a generic concept of "network health."

For example:

```text
interface is UP
```

does not prove external connectivity.

Likewise:

```text
route exists
```

does not prove that packets can reach their destination.

And:

```text
RX/TX counters are increasing
```

does not by itself prove that the intended application-level network path is healthy.

Network health therefore requires an explicit proposition.

Examples of distinct propositions include:

```text
The interface is administratively/configurationally available.
```

```text
A route to a particular destination exists.
```

```text
The host has a usable address on an interface.
```

```text
A specific endpoint is reachable.
```

These are different assessments and require different evidence.

Hostcheck currently observes network configuration and counters. It does not claim to provide general connectivity or reachability assessment.

---

## Network State Semantics

Linux network state requires particular care.

An interface can report:

```text
operstate = unknown
```

without that value alone establishing that the interface is broken.

Loopback is a common example where traditional physical-interface assumptions do not apply.

Similarly, an interface without speed or duplex information is not automatically unhealthy.

Health evaluation must interpret network observations according to their actual Linux semantics rather than applying generic physical-link assumptions to every interface.

---

## Routing Is Configuration Evidence

Hostcheck's route collection observes routing configuration exposed by Linux.

A route represents information used by the kernel's routing system.

It does not, by itself, establish successful packet delivery.

Therefore:

```text
route exists
```

should be treated as routing evidence.

It should not automatically become:

```text
network healthy
```

or:

```text
destination reachable
```

A future reachability assessment would require additional evidence appropriate to the proposition being tested.

---

## Partial Snapshots

Hostcheck intentionally allows a snapshot to contain partial information.

For example:

```text
CPU        available
Memory     available
Filesystem available
Processes  available
Network    unavailable
```

This is still a useful snapshot.

The health layer should not discard the observations that were successfully collected merely because one subsystem failed.

Instead, each assessment should explicitly reflect whether its required evidence is available.

Conceptually:

```text
Memory
  assessable

Filesystem
  assessable

Network
  no health rule currently defined
```

When a concrete health rule exists but its required observation is unavailable, the result is:

```text
Availability: Unassessable
```

This is preferable to converting the entire snapshot into an unhealthy result.

---

## Subsystem Participation

Not every subsystem necessarily participates in every overall health conclusion.

For example, process observations may be useful for diagnostics without being required for the basic host health result.

Likewise, network health may depend on whether network health is relevant to the particular deployment.

This means that overall host health requires explicit participation semantics.

The implementation must not silently assume:

```text
every subsystem is required
```

or:

```text
the worst subsystem status is always the host status
```

Both are policy decisions.

The relationship between subsystem assessments and overall host assessment must therefore be documented before implementation.

---

## Snapshot Evaluation

The current snapshot evaluator provides the first integration boundary between individual health rules and the host snapshot.

Conceptually:

```text
Host Snapshot
      ↓
Snapshot Policy
      ↓
Memory assessment
      +
Filesystem assessment
      ↓
[]Assessment
```

The evaluator currently uses:

```go
type SnapshotPolicy struct {
    Memory     MemoryPolicy
    Filesystem FilesystemPolicy
}
```

The evaluator validates the supplied policy before producing assessments.

Therefore:

```text
invalid policy
    ↓
evaluation error
```

For a valid policy:

```text
missing memory observation
    ↓
memory = Unassessable
```

```text
missing filesystem observation
    ↓
filesystem = Unassessable
```

```text
valid memory observation
    ↓
OK / DEGRADED / CRITICAL
```

```text
valid filesystem observation
    ↓
OK / DEGRADED / CRITICAL
```

The snapshot evaluator does not combine these results into one host status.

This is deliberate.

The project has not yet established sufficient semantics to justify rules such as:

```text
worst status wins
```

or:

```text
any unassessable subsystem makes the host unassessable
```

Those are policy decisions that require explicit participation semantics.

---

## Overall Host Assessment

The overall host assessment is the most sensitive part of the health model because it can hide uncertainty if designed carelessly.

Consider:

```text
Memory     OK
Filesystem DEGRADED
Network    Unassessable
Process    informational
```

Several interpretations are possible.

### Strict interpretation

If any required subsystem cannot be assessed, the overall host assessment is incomplete.

This approach prioritizes evidence completeness.

### Evidence-based interpretation

Available subsystem assessments contribute to the overall result while the result explicitly records incomplete coverage.

This approach preserves useful information without pretending that every subsystem was evaluated.

### Policy-driven interpretation

Configuration determines which subsystems are required and which are optional.

For example:

```text
Memory      required

Filesystem  required

Network     optional

Process     informational
```

Under this model, an unavailable optional subsystem does not necessarily prevent an overall assessment.

These semantics are not interchangeable.

Hostcheck must explicitly choose and document the V1 model rather than silently implementing a "worst status wins" rule.

Until that decision is made, subsystem assessments should remain independent.

---

## Health Must Preserve Uncertainty

The health layer must not turn uncertainty into certainty.

Examples:

```text
collection failed
```

should not become:

```text
unhealthy
```

Likewise:

```text
insufficient samples
```

should not become:

```text
CPU healthy
```

And:

```text
route exists
```

should not become:

```text
network reachable
```

The health result should communicate the limits of the evidence available to it.

This is especially important for a monitoring and reliability tool because false certainty can be more damaging than an explicit lack of information.

---

## Current Exclusions

The following are intentionally outside the current V1 health contract.

### Arbitrary resource thresholds

Hostcheck does not establish universal thresholds such as:

```text
CPU > 90% = CRITICAL

Memory < 10% = CRITICAL

Disk < 10% = CRITICAL
```

The current memory and filesystem rules accept thresholds from the caller.

These values are policy, not Linux facts.

### Connectivity claims

Network configuration does not automatically prove connectivity.

Hostcheck does not currently claim application-level reachability based solely on addresses, routes, or interface state.

### Remediation

Health evaluation does not perform actions.

There is no automatic:

* process termination
* service restart
* route modification
* filesystem cleanup
* interface reset
* resource remediation

### Historical conclusions

A single snapshot should not be used to make historical claims that require a time series.

For example, a snapshot cannot establish that:

```text
memory usage has been increasing continuously
```

unless historical observations are available.

### Performance trends

Trend detection requires repeated observations and historical state.

It is therefore separate from the single-snapshot health boundary.

### Application health

Hostcheck currently evaluates host-level evidence.

It does not automatically establish:

* database health
* HTTP availability
* application correctness
* service-level objectives
* user-facing latency
* external dependency health

Those require application-specific evidence.

---

## Snapshot Timing

The host snapshot records an observation timestamp when collection begins.

That timestamp represents the beginning of host collection.

It does not imply that every subsystem was observed at exactly the same instant.

Collection is currently sequential.

Conceptually:

```text
T0  host snapshot begins

T1  CPU collected

T2  memory collected

T3  filesystem collected

T4  processes collected

T5  network collected

T6  host collection completes
```

The current `Snapshot.ObservedAt` therefore represents the host observation boundary rather than an atomic system-wide instant.

The network subsystem also retains its own observation timestamp because network collection has its own collection boundary.

Health evaluation must respect these semantics.

It must not describe the snapshot as an atomic representation of the entire host.

---

## Live-System Races

Linux is a live system.

Values can change while Hostcheck is reading them.

Examples include:

* processes appearing or disappearing
* process state changing
* CPU counters advancing
* network counters changing
* routes changing
* interfaces changing state

A successful read therefore does not imply that all values across the snapshot represent one perfectly synchronized system state.

The snapshot is an observation assembled from multiple Linux interfaces over a bounded period.

Health evaluation must operate within those semantics.

---

## Health Evaluation Pipeline

The implemented architecture is currently:

```text
Linux
  ↓
Subsystem collection
  ↓
Host Snapshot
  ↓
Derived observations
  ↓
Health policy validation
  ↓
Subsystem health evaluation
  ↓
Subsystem assessments
```

The future overall host assessment remains separate:

```text
Subsystem assessments
  ↓
Defined participation semantics
  ↓
Overall host assessment
```

Each layer has a distinct responsibility.

### Linux interfaces

Provide operating-system state.

### Subsystem collectors

Read, parse, validate, and derive subsystem observations.

### Host snapshot

Combines those observations into one host-level observation boundary and preserves collection errors.

### Derived observation layer

Calculates values that can be derived from collected Linux data.

Examples include:

```text
Memory AvailablePercent
Filesystem AvailablePercent
CPU utilization
```

A derived observation remains an observation. It does not become a health status automatically.

### Health evaluation

Determines whether the available observations are sufficient for defined health rules and evaluates those rules against explicit policy.

### Overall assessment

Will eventually combine subsystem assessments according to explicit participation semantics.

No layer should silently absorb the responsibilities of another.

---

## Health Rules

A health rule should define at least:

```text
Subject

Required evidence

Assessment condition

Policy boundary

Result

Reason

Evidence
```

Conceptually:

```text
Rule

├── subject
├── required evidence
├── applicability
├── evaluation
├── status
└── explanation
```

For example, the current filesystem capacity rule defines:

```text
Subject:
    filesystem

Required evidence:
    available capacity

Policy:
    configured capacity boundaries

Evaluation:
    compare observed capacity with policy

Result:
    OK / DEGRADED / CRITICAL

Evidence:
    observed available percentage
```

The rule is not embedded inside the filesystem collector.

The same separation exists for memory.

---

## Thresholds Must Be Explicit Policy

Thresholds are not facts discovered from Linux.

They are decisions about how observations should be interpreted.

Therefore, a threshold should be:

* explicit
* documented
* configurable where appropriate
* associated with a specific health rule
* tied to a clearly defined observation
* tested

Avoid hidden constants such as:

```go
if availablePercent < 10 {
    return Critical
}
```

without documenting why that boundary exists and what it means.

A health policy should be understandable without reading implementation details.

---

## Health Results Should Be Explainable

A Hostcheck health result should allow an operator to understand why it exists.

For example:

```text
Filesystem

Status: DEGRADED

Reason:
Available capacity is below the configured policy boundary.

Evidence:
available_percent=18.50
```

This is preferable to:

```text
Filesystem: DEGRADED
```

because the latter forces the operator to inspect unrelated implementation details to understand the conclusion.

Explainability also makes testing easier.

A test can verify not only that a status was produced, but that it was produced from the intended evidence.

---

## Health Is Not Remediation

Hostcheck's health layer should report conclusions.

It should not decide what action an operator must take.

For example:

```text
Filesystem DEGRADED
```

is a health assessment.

```text
Delete old logs
```

is an operational recommendation.

```text
Automatically delete old logs
```

is remediation.

These are separate concerns.

Keeping them separate allows Hostcheck to remain useful in environments where remediation is controlled by another system.

---

## Testing Requirements

Health evaluation must be deterministic.

Tests should provide explicit snapshots or controlled observations rather than depending on the current machine's live state.

The health layer should therefore be testable with scenarios such as:

```text
Complete evidence

Partial evidence

Missing evidence

Invalid observation

Collection failure

Insufficient samples

Boundary values

Explicit policy conditions

Invalid policy
```

Tests should verify both:

1. the health result
2. the reason and evidence supporting that result

A health test should not require a particular host's current memory size, filesystem capacity, process list, or network configuration.

Live-system tests remain useful for collectors.

Health-policy tests should be deterministic.

---

## Example Assessment Scenarios

### Complete observations

```text
Memory     assessable
Filesystem assessable
```

Health evaluation can evaluate the rules for which sufficient evidence exists.

The current snapshot evaluator produces independent assessments for both subsystems.

---

### Network collection failure

```text
Memory     assessable
Filesystem assessable
Network    collection failed
```

The memory and filesystem assessments can still be evaluated.

Network evidence remains unavailable.

The network should not automatically be reported as unhealthy.

---

### CPU without sufficient sampling

```text
CPU counters available

CPU utilization unavailable
```

The health layer should not invent utilization from one cumulative sample.

The CPU rule should remain unassessable if utilization is required and the necessary evidence does not exist.

---

### Filesystem capacity available but inode exhaustion possible

```text
Available bytes: high

Available inodes: low
```

The filesystem assessment must not reduce both observations to one capacity value.

Capacity and inode availability are different resource dimensions.

The current filesystem health rule evaluates available block capacity. Inode health remains a separate future rule.

---

### Memory observation unavailable

```text
Memory observation: unavailable
```

The current memory evaluator produces:

```text
Subject: memory

Availability: unassessable

Status:

Reason:
memory observation is unavailable
```

This does not become:

```text
Memory = CRITICAL
```

because the absence of evidence is not evidence of unhealthy memory.

---

### Interface state available

```text
Interface: enp0s3

State: UP

Address: configured

Route: configured
```

These observations establish network configuration information.

They do not automatically establish external connectivity.

---

## Current V1 Boundary

The current V1 health boundary is:

```text
Host Snapshot

    ↓

Determine available evidence

    ↓

Validate explicit health policy

    ↓

Determine whether a health rule is assessable

    ↓

Evaluate the health rule

    ↓

Produce explainable subsystem assessment

    ↓

Keep subsystem assessments independent
```

The current implementation has established:

* explicit assessment availability
* explicit health status
* assessment subjects
* assessment reasons
* retained evidence
* assessment validation invariants
* explicit memory policy
* explicit filesystem policy
* memory available-capacity evaluation
* filesystem available-capacity evaluation
* snapshot-level evaluation of those rules
* invalid policy as an evaluation error
* missing or invalid observation as `Unassessable`

The following remain deliberately unresolved:

* overall host participation rules
* overall host status aggregation
* universal health thresholds
* configurable policy representation beyond current subsystem policies
* structured health output
* CLI presentation
* persistence of historical assessments
* trend evaluation
* alerting

These are implementation decisions that should follow the semantic contract rather than define it accidentally.

---

## Design Principles

The health layer follows several principles.

### 1. Observation before interpretation

Collect what Linux exposes before deciding what it means.

### 2. Assessment requires evidence

Do not produce a health conclusion when the required evidence is unavailable.

### 3. Collection failure is not automatically unhealthy

An inability to observe a subsystem is different from evidence that the subsystem is unhealthy.

### 4. Collection errors are not health statuses

A collection error describes Hostcheck's observation capability.

It should not automatically become `DEGRADED` or `CRITICAL`.

### 5. Preserve partial information

A failed subsystem should not invalidate successfully collected observations.

### 6. Keep policy separate from collection

Linux collectors should not contain arbitrary health thresholds.

### 7. Invalid policy is different from missing evidence

A malformed policy should produce an evaluation error.

Insufficient observation evidence should produce an unassessable assessment.

### 8. Preserve uncertainty

The system should explicitly communicate when a conclusion cannot be established.

### 9. Make conclusions explainable

Every health result should be traceable to evidence and an explicit rule.

### 10. Avoid false precision

A single snapshot should not be used to make conclusions that require historical data.

### 11. Respect Linux semantics

Health rules must operate on what Linux fields actually mean rather than convenient assumptions.

### 12. Keep the architecture small

The health layer should solve the actual evaluation problem without introducing generic abstractions that are not yet justified.

---

## V1 Limitations

The first health implementation will intentionally be limited.

Hostcheck will not initially attempt to solve every aspect of host reliability.

In particular, V1 will not automatically provide:

* universal health thresholds
* application health
* end-to-end connectivity testing
* historical trend analysis
* distributed health aggregation
* automatic remediation
* alert routing
* service-level objectives
* predictive failure detection

The current implementation also does not provide an overall host health verdict.

These can be added later when their evidence requirements and semantics are understood.

The current priority is establishing a correct boundary between observation and health evaluation.

---

## Next Engineering Step

The health contract has now been implemented against the `host.Snapshot` model.

The current Go health model defines:

* assessment availability
* health status
* assessment subjects
* reasons
* retained evidence
* validation invariants
* explicit memory policy
* explicit filesystem policy
* snapshot-level evaluation

The first concrete health rules now cover memory available capacity and filesystem available capacity.

Memory availability is derived from the Linux memory observation and evaluated against caller-supplied policy.

Filesystem available capacity is derived from filesystem statistics and evaluated against caller-supplied policy.

CPU, process, and network observations remain outside concrete health evaluation rules. Their observations are available to the system, but the project has not yet established sufficient evidence and explicit policy to make additional health claims.

The next engineering step is to determine which additional health propositions can be justified from the existing snapshot model.

That review should consider:

1. Which observations are sufficient for health assessment?
2. Which observations require multiple samples?
3. Which subsystem rules are valid for a single snapshot?
4. Which failures should make an assessment unassessable?
5. Which subsystem assessments participate in overall host health?
6. What does an incomplete overall assessment mean?
7. Which thresholds, if any, are justified for V1?
8. Which policies should be configurable?
9. What evidence must every assessment retain?
10. What should the public CLI or JSON representation expose?

The implementation should continue to follow the smallest defensible model.

---

## Summary

Hostcheck now has a clear distinction between collecting host observations and evaluating host health.

The host snapshot is the integration boundary.

Subsystem packages own Linux-specific observation and interpretation.

Derived observation functions calculate values that can be established from those observations.

The health layer consumes the snapshot and determines what can be concluded from the available evidence under explicit policy.

The current implementation can independently assess memory and filesystem available capacity.

The central rule is simple:

> Hostcheck should never claim more than the snapshot can prove.

That means:

```text
Observed ≠ Healthy

Collected ≠ Healthy

Collection failure ≠ Unhealthy

Missing evidence ≠ Unhealthy

Invalid policy ≠ Unhealthy

Configuration ≠ Connectivity

Single snapshot ≠ Historical trend

Derived observation ≠ Health policy
```

The health model now provides a concrete foundation for explicit, deterministic assessment rules.

Memory and filesystem capacity evaluation are the first implemented rules.

No universal Hostcheck health thresholds have been established.

No overall host verdict has been established.

Further health rules should be introduced only when the available evidence, assessment semantics, and policy are explicit enough to justify the conclusion.
