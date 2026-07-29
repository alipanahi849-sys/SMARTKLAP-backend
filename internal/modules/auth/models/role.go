package models

const (
	RoleAdmin     = "admin"
	RoleClubAdmin = "club_admin"
	RoleModerator = "moderator"
	RoleUser      = "user"
)

var DefaultRoles = []string{
	RoleAdmin,
	RoleClubAdmin,
	RoleModerator,
	RoleUser,
}
