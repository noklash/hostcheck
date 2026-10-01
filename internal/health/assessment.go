package health

type Availability string

const (
	Assessable   Availability = "assessable"
	Unassessable Availability = "unassessable"
)

type Status string

const (
	OK       Status = "ok"
	Degraded Status = "degraded"
	Critical Status = "critical"
)

type Assessment struct {
	Subject      string
	Availability Availability
	Status       Status
	Reason       string
	Evidence     []string
}
