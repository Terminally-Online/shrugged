package models

type AccountStatus string

const (
	AccountStatusActive AccountStatus = "active"
	AccountStatusSuspended AccountStatus = "suspended"
	AccountStatusPendingVerification AccountStatus = "pending_verification"
	AccountStatusDeleted AccountStatus = "deleted"
)
