package health

import (
	"errors"
	"net"
	"testing"

	"github.com/noklash/hostcheck/internal/filesystem"
	"github.com/noklash/hostcheck/internal/host"
	"github.com/noklash/hostcheck/internal/memory"
	"github.com/noklash/hostcheck/internal/network"
	"github.com/noklash/hostcheck/internal/process"
	"golang.org/x/sys/unix"
)

var errTestCollection = errors.New("test collection error")

func TestEvaluateSnapshot(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		FilesystemInode: FilesystemInodePolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
	}

	snapshot := host.Snapshot{
		Memory: &memory.MemInfo{
			Total:     1000,
			Available: 500,
		},
		Filesystem: &filesystem.Stats{
			BlocksTotal:     1000,
			BlocksAvailable: 500,
			InodesTotal:     1000,
			InodesFree:      500,
		},
	}

	assessments, err := EvaluateSnapshot(snapshot, policy)
	if err != nil {
		t.Fatalf("EvaluateSnapshot() error = %v", err)
	}

	if len(assessments) != 3 {
		t.Fatalf(
			"len(assessments) = %d, want 3",
			len(assessments),
		)
	}

	if assessments[0].Subject != "memory" {
		t.Fatalf(
			"assessments[0].Subject = %q, want %q",
			assessments[0].Subject,
			"memory",
		)
	}

	if assessments[0].Status != OK {
		t.Fatalf(
			"assessments[0].Status = %q, want %q",
			assessments[0].Status,
			OK,
		)
	}

	if assessments[1].Subject != "filesystem" {
		t.Fatalf(
			"assessments[1].Subject = %q, want %q",
			assessments[1].Subject,
			"filesystem",
		)
	}

	if assessments[1].Status != OK {
		t.Fatalf(
			"assessments[1].Status = %q, want %q",
			assessments[1].Status,
			OK,
		)
	}

	if assessments[2].Subject != "filesystem_inodes" {
		t.Fatalf(
			"assessments[2].Subject = %q, want %q",
			assessments[2].Subject,
			"filesystem_inodes",
		)
	}

	if assessments[2].Status != OK {
		t.Fatalf(
			"assessments[2].Status = %q, want %q",
			assessments[2].Status,
			OK,
		)
	}

	for _, assessment := range assessments {
		if err := assessment.Validate(); err != nil {
			t.Fatalf(
				"Assessment.Validate() error = %v",
				err,
			)
		}
	}
}

func TestEvaluateSnapshotMarksMissingObservationsUnassessable(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		FilesystemInode: FilesystemInodePolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
	}

	assessments, err := EvaluateSnapshot(
		host.Snapshot{},
		policy,
	)
	if err != nil {
		t.Fatalf(
			"EvaluateSnapshot() error = %v",
			err,
		)
	}

	if len(assessments) != 3 {
		t.Fatalf(
			"len(assessments) = %d, want 3",
			len(assessments),
		)
	}

	for _, assessment := range assessments {
		if assessment.Availability != Unassessable {
			t.Fatalf(
				"assessment %q Availability = %q, want %q",
				assessment.Subject,
				assessment.Availability,
				Unassessable,
			)
		}

		if assessment.Status != "" {
			t.Fatalf(
				"assessment %q Status = %q, want empty status",
				assessment.Subject,
				assessment.Status,
			)
		}

		if err := assessment.Validate(); err != nil {
			t.Fatalf(
				"Assessment.Validate() error = %v",
				err,
			)
		}
	}
}

func TestEvaluateSnapshotPreservesCollectionErrors(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		FilesystemInode: FilesystemInodePolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
	}

	snapshot := host.Snapshot{
		Errors: []host.CollectionError{
			{
				Subsystem: "memory",
				Err:       errTestCollection,
			},
			{
				Subsystem: "filesystem",
				Err:       errTestCollection,
			},
		},
	}

	assessments, err := EvaluateSnapshot(snapshot, policy)
	if err != nil {
		t.Fatalf(
			"EvaluateSnapshot() error = %v",
			err,
		)
	}

	if len(assessments) != 3 {
		t.Fatalf(
			"len(assessments) = %d, want 3",
			len(assessments),
		)
	}

	if assessments[0].Reason != "memory collection failed: test collection error" {
		t.Fatalf(
			"memory Reason = %q, want collection failure reason",
			assessments[0].Reason,
		)
	}

	if assessments[1].Reason != "filesystem collection failed: test collection error" {
		t.Fatalf(
			"filesystem Reason = %q, want collection failure reason",
			assessments[1].Reason,
		)
	}

	if assessments[2].Availability != Unassessable {
		t.Fatalf(
			"filesystem inode Availability = %q, want %q",
			assessments[2].Availability,
			Unassessable,
		)
	}
}

func TestEvaluateSnapshotSkipsUnconfiguredProcessAndNetworkChecks(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		FilesystemInode: FilesystemInodePolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
	}

	snapshot := host.Snapshot{
		Memory: &memory.MemInfo{
			Total:     1000,
			Available: 500,
		},
		Filesystem: &filesystem.Stats{
			BlocksTotal:     1000,
			BlocksAvailable: 500,
			InodesTotal:     1000,
			InodesFree:      500,
		},
		Processes: []process.Process{},
		Network:   &network.Network{},
	}

	assessments, err := EvaluateSnapshot(snapshot, policy)
	if err != nil {
		t.Fatalf(
			"EvaluateSnapshot() error = %v",
			err,
		)
	}

	if len(assessments) != 3 {
		t.Fatalf(
			"len(assessments) = %d, want 3",
			len(assessments),
		)
	}

	for _, assessment := range assessments {
		if assessment.Subject == "process_state" {
			t.Fatal("unexpected process_state assessment")
		}

		if assessment.Subject == "network_interface" {
			t.Fatal("unexpected network_interface assessment")
		}

		if assessment.Subject == "network_route" {
			t.Fatal("unexpected network_route assessment")
		}
	}
}

func TestEvaluateSnapshotEvaluatesConfiguredProcessState(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		FilesystemInode: FilesystemInodePolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		ProcessState: &ProcessStatePolicy{
			DegradedAtOrAbove: 1,
			CriticalAtOrAbove: 3,
		},
	}

	snapshot := host.Snapshot{
		Memory: &memory.MemInfo{
			Total:     1000,
			Available: 500,
		},
		Filesystem: &filesystem.Stats{
			BlocksTotal:     1000,
			BlocksAvailable: 500,
			InodesTotal:     1000,
			InodesFree:      500,
		},
		Processes: []process.Process{
			{
				Stats: process.Stats{
					State: 'D',
				},
			},
		},
	}

	assessments, err := EvaluateSnapshot(snapshot, policy)
	if err != nil {
		t.Fatalf(
			"EvaluateSnapshot() error = %v",
			err,
		)
	}

	if len(assessments) != 4 {
		t.Fatalf(
			"len(assessments) = %d, want 4",
			len(assessments),
		)
	}

	processAssessment := assessments[3]

	if processAssessment.Subject != "process_state" {
		t.Fatalf(
			"process assessment Subject = %q, want %q",
			processAssessment.Subject,
			"process_state",
		)
	}

	if processAssessment.Status != Degraded {
		t.Fatalf(
			"process assessment Status = %q, want %q",
			processAssessment.Status,
			Degraded,
		)
	}

	if processAssessment.Availability != Assessable {
		t.Fatalf(
			"process assessment Availability = %q, want %q",
			processAssessment.Availability,
			Assessable,
		)
	}
}

func TestEvaluateSnapshotMarksProcessCollectionFailureUnassessable(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		FilesystemInode: FilesystemInodePolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		ProcessState: &ProcessStatePolicy{
			DegradedAtOrAbove: 1,
			CriticalAtOrAbove: 3,
		},
	}

	snapshot := host.Snapshot{
		Memory: &memory.MemInfo{
			Total:     1000,
			Available: 500,
		},
		Filesystem: &filesystem.Stats{
			BlocksTotal:     1000,
			BlocksAvailable: 500,
			InodesTotal:     1000,
			InodesFree:      500,
		},
		Errors: []host.CollectionError{
			{
				Subsystem: "process",
				Err:       errTestCollection,
			},
		},
	}

	assessments, err := EvaluateSnapshot(snapshot, policy)
	if err != nil {
		t.Fatalf(
			"EvaluateSnapshot() error = %v",
			err,
		)
	}

	if len(assessments) != 4 {
		t.Fatalf(
			"len(assessments) = %d, want 4",
			len(assessments),
		)
	}

	processAssessment := assessments[3]

	if processAssessment.Subject != "process_state" {
		t.Fatalf(
			"process assessment Subject = %q, want %q",
			processAssessment.Subject,
			"process_state",
		)
	}

	if processAssessment.Availability != Unassessable {
		t.Fatalf(
			"process assessment Availability = %q, want %q",
			processAssessment.Availability,
			Unassessable,
		)
	}

	if processAssessment.Status != "" {
		t.Fatalf(
			"process assessment Status = %q, want empty status",
			processAssessment.Status,
		)
	}

	if processAssessment.Reason != "process collection failed: test collection error" {
		t.Fatalf(
			"process assessment Reason = %q, want collection failure reason",
			processAssessment.Reason,
		)
	}
}

func TestEvaluateSnapshotEvaluatesConfiguredNetworkInterface(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		FilesystemInode: FilesystemInodePolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		NetworkInterfaces: []NetworkInterfacePolicy{
			{
				Name:           "enp0s3",
				RequireCarrier: true,
			},
		},
	}

	snapshot := host.Snapshot{
		Memory: &memory.MemInfo{
			Total:     1000,
			Available: 500,
		},
		Filesystem: &filesystem.Stats{
			BlocksTotal:     1000,
			BlocksAvailable: 500,
			InodesTotal:     1000,
			InodesFree:      500,
		},
		Network: &network.Network{
			Interfaces: []network.Interface{
				{
					Name:      "enp0s3",
					OperState: "up",
					Carrier:   true,
				},
			},
		},
	}

	assessments, err := EvaluateSnapshot(snapshot, policy)
	if err != nil {
		t.Fatalf(
			"EvaluateSnapshot() error = %v",
			err,
		)
	}

	if len(assessments) != 4 {
		t.Fatalf(
			"len(assessments) = %d, want 4",
			len(assessments),
		)
	}

	networkAssessment := assessments[3]

	if networkAssessment.Subject != "network_interface" {
		t.Fatalf(
			"network assessment Subject = %q, want %q",
			networkAssessment.Subject,
			"network_interface",
		)
	}

	if networkAssessment.Availability != Assessable {
		t.Fatalf(
			"network assessment Availability = %q, want %q",
			networkAssessment.Availability,
			Assessable,
		)
	}

	if networkAssessment.Status != OK {
		t.Fatalf(
			"network assessment Status = %q, want %q",
			networkAssessment.Status,
			OK,
		)
	}
}

func TestEvaluateSnapshotMarksNetworkCollectionFailureUnassessable(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		FilesystemInode: FilesystemInodePolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		NetworkInterfaces: []NetworkInterfacePolicy{
			{
				Name: "enp0s3",
			},
		},
	}

	snapshot := host.Snapshot{
		Memory: &memory.MemInfo{
			Total:     1000,
			Available: 500,
		},
		Filesystem: &filesystem.Stats{
			BlocksTotal:     1000,
			BlocksAvailable: 500,
			InodesTotal:     1000,
			InodesFree:      500,
		},
		Errors: []host.CollectionError{
			{
				Subsystem: "network",
				Err:       errTestCollection,
			},
		},
	}

	assessments, err := EvaluateSnapshot(snapshot, policy)
	if err != nil {
		t.Fatalf(
			"EvaluateSnapshot() error = %v",
			err,
		)
	}

	if len(assessments) != 4 {
		t.Fatalf(
			"len(assessments) = %d, want 4",
			len(assessments),
		)
	}

	networkAssessment := assessments[3]

	if networkAssessment.Subject != "network_interface" {
		t.Fatalf(
			"network assessment Subject = %q, want %q",
			networkAssessment.Subject,
			"network_interface",
		)
	}

	if networkAssessment.Availability != Unassessable {
		t.Fatalf(
			"network assessment Availability = %q, want %q",
			networkAssessment.Availability,
			Unassessable,
		)
	}

	if networkAssessment.Status != "" {
		t.Fatalf(
			"network assessment Status = %q, want empty status",
			networkAssessment.Status,
		)
	}

	if networkAssessment.Reason != "network collection failed: test collection error" {
		t.Fatalf(
			"network assessment Reason = %q, want collection failure reason",
			networkAssessment.Reason,
		)
	}
}

func TestEvaluateSnapshotEvaluatesConfiguredNetworkRoutes(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		FilesystemInode: FilesystemInodePolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		NetworkRoutes: []NetworkRoutePolicy{
			{
				Family:        unix.AF_INET,
				Destination:   nil,
				PrefixLen:     0,
				InterfaceName: "enp0s3",
			},
			{
				Family:        unix.AF_INET,
				Destination:   net.IPv4(8, 8, 8, 8),
				PrefixLen:     32,
				InterfaceName: "enp0s3",
			},
		},
	}

	snapshot := host.Snapshot{
		Memory: &memory.MemInfo{
			Total:     1000,
			Available: 500,
		},
		Filesystem: &filesystem.Stats{
			BlocksTotal:     1000,
			BlocksAvailable: 500,
			InodesTotal:     1000,
			InodesFree:      500,
		},
		Network: &network.Network{
			Interfaces: []network.Interface{
				{
					Index: 2,
					Name:  "enp0s3",
				},
			},
			Routes: []network.Route{
				{
					Family:         unix.AF_INET,
					Destination:    nil,
					PrefixLen:      0,
					InterfaceIndex: 2,
				},
				{
					Family:         unix.AF_INET,
					Destination:    net.IPv4(8, 8, 8, 8),
					PrefixLen:      32,
					InterfaceIndex: 2,
				},
			},
		},
	}

	assessments, err := EvaluateSnapshot(snapshot, policy)
	if err != nil {
		t.Fatalf(
			"EvaluateSnapshot() error = %v",
			err,
		)
	}

	if len(assessments) != 5 {
		t.Fatalf(
			"len(assessments) = %d, want 5",
			len(assessments),
		)
	}

	for i := 3; i < 5; i++ {
		if assessments[i].Subject != "network_route" {
			t.Fatalf(
				"assessments[%d].Subject = %q, want %q",
				i,
				assessments[i].Subject,
				"network_route",
			)
		}

		if assessments[i].Availability != Assessable {
			t.Fatalf(
				"assessments[%d].Availability = %q, want %q",
				i,
				assessments[i].Availability,
				Assessable,
			)
		}

		if assessments[i].Status != OK {
			t.Fatalf(
				"assessments[%d].Status = %q, want %q",
				i,
				assessments[i].Status,
				OK,
			)
		}
	}
}

func TestEvaluateSnapshotRejectsInvalidPolicy(t *testing.T) {
	policy := SnapshotPolicy{
		Memory: MemoryPolicy{
			DegradedBelowPercent: 10,
			CriticalBelowPercent: 20,
		},
		Filesystem: FilesystemPolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
		FilesystemInode: FilesystemInodePolicy{
			DegradedBelowPercent: 20,
			CriticalBelowPercent: 10,
		},
	}

	if _, err := EvaluateSnapshot(host.Snapshot{}, policy); err == nil {
		t.Fatal(
			"EvaluateSnapshot() error = nil, want invalid policy error",
		)
	}
}
