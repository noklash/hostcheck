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

---

## Assessment Status

Health assessment requires a status vocabulary that does not confuse health state with observability.

The proposed V1 model separates two concepts.

### Availability

Availability describes whether sufficient evidence exists to perform a particular assessment.

Possible states are:

```text
Assessable
Unassessable
```

### Status

Status describes the health conclusion when sufficient evidence exists.

Possible V1 states are:

```text
OK
DEGRADED
CRITICAL
```

These concepts should not be collapsed into one `UNKNOWN` value.

For example:

```text
Network
Availability: Unassessable
Reason: collection failed
```

is different from:

```text
Network
Availability: Assessable
Status: DEGRADED
Reason: explicit health policy determined the observed condition
```

The first describes an evidence problem.

The second describes a health condition.

That distinction is important because treating both as `UNKNOWN` would hide why the assessment could not or did conclude something.

---

## Evidence

A health assessment should be explainable from the observations that produced it.

A useful conceptual model is:

```text
Assessment
├── subject
├── availability
├── status
├── reason
└── evidence
```

The exact Go representation can be refined during implementation.

The important requirement is that a health result should not become an opaque label.

For example, a future filesystem assessment might conceptually contain:

```text
Subject: filesystem
Availability: assessable
Status: degraded
Reason: available capacity crossed configured policy boundary
Evidence:
    available bytes
    available percentage
    filesystem path
```

The evidence belongs to the assessment because it allows the result to be understood and later inspected.

The health layer should not replace the underlying observation with the status.

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

## Current Health Inputs

Not every Hostcheck observation is currently eligible to produce a health assessment.

The current V1 boundary is intentionally conservative.

### CPU

Available observations include CPU accounting counters and derived utilization where sufficient samples exist.

CPU counters themselves are not a health verdict.

A future CPU health rule may use derived utilization or other explicit evidence, but the rule must define:

* required observations
* required sampling relationship
* calculation semantics
* policy boundaries

A single CPU snapshot should not be treated as sufficient evidence for utilization-based health.

---

### Memory

Hostcheck collects Linux memory information from `/proc/meminfo`.

Memory observations include values such as:

* total memory
* available memory
* free memory
* buffers
* cached memory
* swap information where exposed

The health layer must use the semantics of these Linux fields rather than treating similarly named values as interchangeable.

In particular, `MemFree` must not automatically be interpreted as the amount of memory available to applications.

`MemAvailable` provides a more meaningful basis for understanding memory availability on modern Linux systems.

However, the existence of a memory observation does not establish a health threshold.

No arbitrary memory percentage is currently defined as `DEGRADED` or `CRITICAL`.

---

### Filesystem

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

However, the health layer must distinguish observation from policy.

For example:

```text
AvailablePercent = 8%
```

does not inherently mean:

```text
DEGRADED
```

The meaning of that value depends on an explicit health policy.

The same applies to inode availability.

A filesystem may have available storage capacity while being constrained by inode exhaustion, or the reverse.

Health evaluation should therefore avoid reducing filesystem health to a single capacity percentage.

---

### Processes

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

---

### Network

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
CPU
  assessable

Memory
  assessable

Filesystem
  assessable

Network
  unassessable
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

## Overall Host Assessment

The overall host assessment is the most sensitive part of the health model because it can hide uncertainty if designed carelessly.

Consider:

```text
CPU        OK
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
CPU         required
Memory      required
Filesystem  required
Network     optional
Process     informational
```

Under this model, an unavailable optional subsystem does not necessarily prevent an overall assessment.

These semantics are not interchangeable.

Hostcheck must explicitly choose and document the V1 model rather than silently implementing a "worst status wins" rule.

Until that decision is made, the host snapshot should remain the stable integration boundary and health evaluation should not invent an overall host verdict.

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

Hostcheck does not currently define universal thresholds such as:

```text
CPU > 90% = CRITICAL
Memory < 10% = CRITICAL
Disk < 10% = CRITICAL
```

These values may eventually exist as policy, but they must have an explicit justification and clear semantics.

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

The intended architecture is:

```text
Linux
  ↓
Subsystem collection
  ↓
Host Snapshot
  ↓
Evidence validation
  ↓
Health evaluation
  ↓
Subsystem assessments
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

### Health evaluation

Determines whether the available observations are sufficient for defined health rules and evaluates those rules.

### Overall assessment

Combines subsystem assessments according to explicit participation semantics.

No layer should silently absorb the responsibilities of another.

---

## Health Rules

A future health rule should define at least:

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

For example, a filesystem rule might eventually define:

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
    observed available capacity
```

The rule should not be embedded inside the filesystem collector.

---

## Thresholds Must Be Explicit Policy

Thresholds are not facts discovered from Linux.

They are decisions about how observations should be interpreted.

Therefore, a threshold should eventually be:

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

A future Hostcheck output should allow an operator to understand why a result exists.

For example:

```text
Filesystem
Status: DEGRADED

Reason:
Available capacity is below the configured policy boundary.

Evidence:
Path: /
Available: ...
AvailablePercent: ...
```

This is preferable to:

```text
Filesystem: DEGRADED
```

because the latter forces the operator to inspect unrelated implementation details to understand the conclusion.

Explainability also makes future testing easier.

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
Collection failure
Insufficient samples
Boundary values
Explicit policy conditions
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
CPU        assessable
Memory     assessable
Filesystem assessable
Network    assessable
```

Health evaluation can evaluate the rules for which sufficient evidence exists.

---

### Network collection failure

```text
CPU        assessable
Memory     assessable
Filesystem assessable
Network    unassessable
```

The network assessment should communicate that network evidence is unavailable.

It should not automatically report the network as unhealthy.

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

The current V1 health boundary is therefore:

```text
Host Snapshot
    ↓
Determine available evidence
    ↓
Determine whether a health rule is assessable
    ↓
Evaluate explicit health policy
    ↓
Produce explainable subsystem assessment
    ↓
Combine assessments only according to defined participation semantics
```

The following remain deliberately unresolved until the semantics are reviewed:

* exact Go assessment types
* exact status representation
* overall host participation rules
* health thresholds
* configurable policy
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

### 4. Preserve partial information

A failed subsystem should not invalidate successfully collected observations.

### 5. Keep policy separate from collection

Linux collectors should not contain arbitrary health thresholds.

### 6. Preserve uncertainty

The system should explicitly communicate when a conclusion cannot be established.

### 7. Make conclusions explainable

Every health result should be traceable to evidence and an explicit rule.

### 8. Avoid false precision

A single snapshot should not be used to make conclusions that require historical data.

### 9. Respect Linux semantics

Health rules must operate on what Linux fields actually mean rather than convenient assumptions.

### 10. Keep the architecture small

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

These can be added later when their evidence requirements and semantics are understood.

The current priority is establishing a correct boundary between observation and health evaluation.

---

````markdown
## Next Engineering Step

The health contract has now been established against the `host.Snapshot` model.

The initial Go health model defines:

* assessment availability
* health status
* assessment subjects
* reasons
* retained evidence
* validation invariants

The first concrete health rule evaluates filesystem available capacity using an explicit caller-supplied policy. The policy defines degraded and critical thresholds without establishing Hostcheck-wide default thresholds.

CPU, memory, process, and network observations currently remain outside concrete health evaluation rules. Their observations are available to the health layer, but the project has not yet established sufficient evidence and explicit policy to make health claims for them.

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

Hostcheck currently has a clear distinction between collecting host observations and evaluating host health.

The host snapshot is the integration boundary.

Subsystem packages own Linux-specific observation and interpretation.

The health layer consumes the snapshot and determines what can be concluded from the available evidence.

The central rule is simple:

> Hostcheck should never claim more than the snapshot can prove.

That means:

```text
Observed ≠ Healthy

Collected ≠ Healthy

Collection failure ≠ Unhealthy

Missing evidence ≠ Unhealthy

Configuration ≠ Connectivity

Single snapshot ≠ Historical trend

Derived observation ≠ Health policy
````

The health model now provides the foundation for explicit, deterministic assessment rules.

The first concrete rule is filesystem available-capacity evaluation using caller-supplied policy. No universal Hostcheck health thresholds have been established.

Further health rules should be introduced only when the available evidence, assessment semantics, and policy are explicit enough to justify the conclusion.