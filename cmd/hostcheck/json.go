package main

import (
	"encoding/json"
	"io"

	"github.com/noklash/hostcheck/internal/health"
	"github.com/noklash/hostcheck/internal/host"
)

type jsonResult struct {
	ObservedAt  string           `json:"observed_at"`
	Status      *health.Status   `json:"status"`
	Coverage    health.Coverage  `json:"coverage"`
	Assessments []jsonAssessment `json:"assessments"`
}

type jsonAssessment struct {
	Subject      string              `json:"subject"`
	Availability health.Availability `json:"availability"`
	Status       *health.Status      `json:"status"`
	Reason       string              `json:"reason"`
	Evidence     []string            `json:"evidence"`
}

func marshalJSONResult(
	snapshot host.Snapshot,
	result health.Result,
) ([]byte, error) {
	output := jsonResult{
		ObservedAt: snapshot.ObservedAt.Format("2006-01-02T15:04:05Z07:00"),
		Coverage:   result.Coverage,
		Assessments: make(
			[]jsonAssessment,
			0,
			len(result.Assessments),
		),
	}

	if result.Status != "" {
		status := result.Status
		output.Status = &status
	}

	for _, assessment := range result.Assessments {
		item := jsonAssessment{
			Subject:      assessment.Subject,
			Availability: assessment.Availability,
			Reason:       assessment.Reason,
			Evidence:     assessment.Evidence,
		}

		if assessment.Status != "" {
			status := assessment.Status
			item.Status = &status
		}

		output.Assessments = append(output.Assessments, item)
	}

	return json.Marshal(output)
}

func printJSONResult(
	w io.Writer,
	snapshot host.Snapshot,
	result health.Result,
) error {
	data, err := marshalJSONResult(snapshot, result)
	if err != nil {
		return err
	}

	_, err = w.Write(append(data, '\n'))
	return err
}
