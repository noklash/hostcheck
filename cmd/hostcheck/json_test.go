package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/noklash/hostcheck/internal/health"
	"github.com/noklash/hostcheck/internal/host"
)

func TestMarshalJSONResult(t *testing.T) {
	observedAt := time.Date(
		2026,
		10,
		8,
		7,
		30,
		0,
		0,
		time.FixedZone("WAT", 60*60),
	)

	snapshot := host.Snapshot{
		ObservedAt: observedAt,
	}

	result := health.Result{
		Status:   health.Degraded,
		Coverage: health.Partial,
		Assessments: []health.Assessment{
			{
				Subject:      "memory",
				Availability: health.Assessable,
				Status:       health.Degraded,
				Reason:       "available memory is below the configured degraded threshold",
				Evidence: []string{
					"available_percent=15.25",
				},
			},
			{
				Subject:      "network_interface",
				Availability: health.Unassessable,
				Reason:       "network observation is unavailable",
			},
		},
	}

	data, err := marshalJSONResult(snapshot, result)
	if err != nil {
		t.Fatalf("marshalJSONResult() error = %v", err)
	}

	var got struct {
		ObservedAt  string `json:"observed_at"`
		Status      string `json:"status"`
		Coverage    string `json:"coverage"`
		Assessments []struct {
			Subject      string   `json:"subject"`
			Availability string   `json:"availability"`
			Status       *string  `json:"status"`
			Reason       string   `json:"reason"`
			Evidence     []string `json:"evidence"`
		} `json:"assessments"`
	}

	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("JSON should be valid: %v", err)
	}

	if got.ObservedAt != "2026-10-08T07:30:00+01:00" {
		t.Fatalf(
			"observed_at = %q, want %q",
			got.ObservedAt,
			"2026-10-08T07:30:00+01:00",
		)
	}

	if got.Status != "degraded" {
		t.Fatalf("status = %q, want degraded", got.Status)
	}

	if got.Coverage != "partial" {
		t.Fatalf("coverage = %q, want partial", got.Coverage)
	}

	if len(got.Assessments) != 2 {
		t.Fatalf(
			"assessment count = %d, want 2",
			len(got.Assessments),
		)
	}

	memory := got.Assessments[0]

	if memory.Subject != "memory" {
		t.Fatalf("first subject = %q, want memory", memory.Subject)
	}

	if memory.Availability != "assessable" {
		t.Fatalf(
			"memory availability = %q, want assessable",
			memory.Availability,
		)
	}

	if memory.Status == nil || *memory.Status != "degraded" {
		t.Fatalf("memory status = %v, want degraded", memory.Status)
	}

	if memory.Reason == "" {
		t.Fatal("memory reason should not be empty")
	}

	if len(memory.Evidence) != 1 ||
		memory.Evidence[0] != "available_percent=15.25" {
		t.Fatalf(
			"memory evidence = %v, want [available_percent=15.25]",
			memory.Evidence,
		)
	}

	network := got.Assessments[1]

	if network.Subject != "network_interface" {
		t.Fatalf(
			"second subject = %q, want network_interface",
			network.Subject,
		)
	}

	if network.Availability != "unassessable" {
		t.Fatalf(
			"network availability = %q, want unassessable",
			network.Availability,
		)
	}

	if network.Status != nil {
		t.Fatalf(
			"unassessable status = %v, want null",
			*network.Status,
		)
	}

	if !strings.Contains(network.Reason, "unavailable") {
		t.Fatalf(
			"network reason = %q, want unavailable explanation",
			network.Reason,
		)
	}
}

func TestMarshalJSONUnavailableResult(t *testing.T) {
	snapshot := host.Snapshot{
		ObservedAt: time.Date(
			2026,
			10,
			8,
			7,
			30,
			0,
			0,
			time.FixedZone("WAT", 60*60),
		),
	}

	result := health.Result{
		Coverage: health.Unavailable,
	}

	data, err := marshalJSONResult(snapshot, result)
	if err != nil {
		t.Fatalf("marshalJSONResult() error = %v", err)
	}

	var got struct {
		Status   *string `json:"status"`
		Coverage string  `json:"coverage"`
	}

	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("JSON should be valid: %v", err)
	}

	if got.Status != nil {
		t.Fatalf("status = %v, want null", *got.Status)
	}

	if got.Coverage != "unavailable" {
		t.Fatalf("coverage = %q, want unavailable", got.Coverage)
	}
}
