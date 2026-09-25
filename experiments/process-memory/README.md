# Process Memory Experiment

This experiment demonstrates the difference between virtual memory and resident memory using Hostcheck's process memory collector.

## Experiment

The program:

1. records the current process memory,
2. creates a 100 MiB anonymous memory mapping,
3. records memory again without touching the mapping,
4. writes to every page in the mapping,
5. records memory again,
6. reports the observed deltas.

The system page size is obtained from the operating system with `os.Getpagesize()`.

## Expected Behavior

After creating the mapping:

```text
virtual memory increases
resident memory increases much less
```

After touching every page:

```text
virtual memory remains approximately unchanged
resident memory increases by approximately the mapping size
anonymous resident memory increases by approximately the mapping size
```

The exact values vary because the Go runtime and the process itself continue allocating and changing state between measurements.

## Run

From the repository root:

```bash
go run ./experiments/process-memory
```

## What This Demonstrates

The experiment provides direct evidence that:

```text
virtual address space
```

and:

```text
resident physical memory
```

are different forms of process memory accounting.

A process can reserve virtual address space without immediately making all of that memory resident.

Writing to previously untouched anonymous pages causes those pages to become resident.

## Limitations

This is a process-level observation experiment, not a benchmark.

It does not attempt to isolate every allocation made by the Go runtime, and it does not measure unique physical ownership of shared pages.

The experiment is intended to demonstrate Linux memory-accounting behavior rather than produce an exact accounting of the Go program's 100 MiB allocation.
