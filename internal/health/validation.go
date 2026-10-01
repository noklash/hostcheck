package health

import "fmt"

func (a Assessment) Validate() error {
	if a.Subject == "" {
		return fmt.Errorf("assessment subject is required")
	}

	switch a.Availability {
	case Assessable:
		switch a.Status {
		case OK, Degraded, Critical:
		default:
			return fmt.Errorf("assessable assessment requires a valid status")
		}

	case Unassessable:
		if a.Status != "" {
			return fmt.Errorf("unassessable assessment must not have a status")
		}

	default:
		return fmt.Errorf("invalid availability %q", a.Availability)
	}

	return nil
}
