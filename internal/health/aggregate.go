package health

import "fmt"

// Coverage describes how completely the supplied health rules were evaluated.
type Coverage string

const (
	Complete    Coverage = "complete"
	Partial     Coverage = "partial"
	Unavailable Coverage = "unavailable"
)

// Result is the host-level result produced from individual health assessments.
//
// Status represents the worst status among assessable assessments.
// Coverage describes whether all, some, or none of the supplied assessments
// were assessable.
//
// A partial result is intentionally possible. For example, a host can be
// critical because memory is critically low while filesystem health could not
// be assessed. The critical condition remains visible without pretending that
// the host was completely evaluated.
type Result struct {
	Status      Status
	Coverage    Coverage
	Assessments []Assessment
}

// Validate validates a host-level health result and its assessments.
func (r Result) Validate() error {
	switch r.Coverage {
	case Complete, Partial:
		if len(r.Assessments) == 0 {
			return fmt.Errorf(
				"%s health result must contain at least one assessment",
				r.Coverage,
			)
		}

		if r.Status != OK && r.Status != Degraded && r.Status != Critical {
			return fmt.Errorf(
				"assessable health result must have a valid status: got %q",
				r.Status,
			)
		}

	case Unavailable:
		if r.Status != "" {
			return fmt.Errorf(
				"unavailable health result must not have a status: got %q",
				r.Status,
			)
		}

	default:
		return fmt.Errorf(
			"invalid health result coverage: %q",
			r.Coverage,
		)
	}

	for i, assessment := range r.Assessments {
		if err := assessment.Validate(); err != nil {
			return fmt.Errorf(
				"assessment %d (%q): %w",
				i,
				assessment.Subject,
				err,
			)
		}
	}

	return nil
}

// Aggregate combines individual assessments into one host-level result.
//
// Critical takes precedence over degraded, and degraded takes precedence over
// OK. Unassessable assessments do not become critical merely because they are
// unavailable.
//
// Coverage is:
//   - complete when every assessment is assessable
//   - partial when at least one assessment is assessable and at least one is
//     unassessable
//   - unavailable when no assessment is assessable
//
// Every assessment is validated before aggregation.
func Aggregate(assessments []Assessment) (Result, error) {
	if len(assessments) == 0 {
		return Result{
			Coverage: Unavailable,
		}, nil
	}

	result := Result{
		Assessments: assessments,
	}

	assessableCount := 0

	for i, assessment := range assessments {
		if err := assessment.Validate(); err != nil {
			return Result{}, fmt.Errorf(
				"assessment %d (%q): %w",
				i,
				assessment.Subject,
				err,
			)
		}

		if assessment.Availability == Unassessable {
			continue
		}

		assessableCount++

		switch {
		case result.Status == "":
			result.Status = assessment.Status

		case assessment.Status == Critical:
			result.Status = Critical

		case assessment.Status == Degraded && result.Status == OK:
			result.Status = Degraded
		}
	}

	switch {
	case assessableCount == 0:
		result.Coverage = Unavailable

	case assessableCount == len(assessments):
		result.Coverage = Complete

	default:
		result.Coverage = Partial
	}

	return result, nil
}
