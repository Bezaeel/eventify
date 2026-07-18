package domain

type AllEntities struct {
	Events          Event
	Permissions     Permission
	RolePermissions RolePermissions
	Roles           Role
	Users           User
	UserRoles       UserRole
}
