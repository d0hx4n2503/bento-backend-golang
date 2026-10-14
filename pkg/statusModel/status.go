// Package statusmodel defines status values shared by the current API model.
package statusmodel

// Status is a type-safe status value.
type Status string

const (
	Pending     Status = "Pending"
	Processed   Status = "Processed"
	Cancelled   Status = "Cancelled"
	UnderReview Status = "UnderReview"
	Overdue     Status = "Overdue"
)
