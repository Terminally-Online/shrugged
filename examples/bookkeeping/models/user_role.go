package models

type UserRole string

const (
	UserRoleAdmin     UserRole = "admin"
	UserRoleModerator UserRole = "moderator"
	UserRoleMember    UserRole = "member"
	UserRoleGuest     UserRole = "guest"
)
