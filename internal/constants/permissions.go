package constants

// PermissionConstants defines all available permissions in the system
type PermissionConstants struct {
	EventPermissions EventPermissions
	UserPermissions  UserPermissions
	AdminPermission  []string
}

// EventPermissions defines permissions related to events
type EventPermissions struct {
	Create string
	Read   string
	Update string
	Delete string
	Admin  string
}

// UserPermissions defines permissions related to users
type UserPermissions struct {
	Create string
	Read   string
	Update string
	Delete string
	Admin  string
}

// Permissions is a global instance of PermissionConstants
var Permissions = PermissionConstants{
	EventPermissions: EventPermissions{
		Create: "events.create",
		Read:   "events.read",
		Update: "events.update",
		Delete: "events.delete",
		Admin:  "events.admin",
	},
	UserPermissions: UserPermissions{
		Create: "users.create",
		Read:   "users.read",
		Update: "users.update",
		Delete: "users.delete",
		Admin:  "users.admin",
	},
	AdminPermission: []string{
		"events.admin",
		"users.admin",
	},
}
