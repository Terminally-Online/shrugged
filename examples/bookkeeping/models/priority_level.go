package models

type PriorityLevel string

const (
	PriorityLevelLow PriorityLevel = "low"
	PriorityLevelMedium PriorityLevel = "medium"
	PriorityLevelHigh PriorityLevel = "high"
	PriorityLevelCritical PriorityLevel = "critical"
)
