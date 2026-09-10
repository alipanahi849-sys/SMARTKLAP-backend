package service

import (
	"context"
	"strings"
	"time"

	authmodels "clap/internal/modules/auth/models"
	authrepo "clap/internal/modules/auth/repository"
	"clap/internal/modules/user/dto"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

var allowedAdminRoles = map[string]struct{}{
	string(utils.RoleAdmin):     {},
	string(utils.RoleClubAdmin): {},
	"moderator":                 {},
	string(utils.RoleUser):      {},
}

var allowedPanelPermissions = map[string]struct{}{}

func init() {
	for _, key := range authmodels.AllPanelPermissions {
		allowedPanelPermissions[key] = struct{}{}
	}
}

// AdminUserService lets admins list, create, and edit panel accounts.
type AdminUserService interface {
	List(ctx context.Context, filters dto.AdminUserListFilters) (*dto.AdminUserListResponse, error)
	Get(ctx context.Context, userID uuid.UUID) (*dto.AdminUserItem, error)
	Create(ctx context.Context, actorID uuid.UUID, req *dto.AdminCreateUserRequest) (*dto.AdminUserItem, error)
	Update(ctx context.Context, actorID, userID uuid.UUID, req *dto.AdminUpdateUserRequest) (*dto.AdminUserItem, error)
}

type adminUserService struct {
	users authrepo.UserRepository
	roles authrepo.RoleRepository
}

// NewAdminUserService constructs the admin user service.
func NewAdminUserService(users authrepo.UserRepository, roles authrepo.RoleRepository) AdminUserService {
	return &adminUserService{users: users, roles: roles}
}

func (s *adminUserService) List(ctx context.Context, filters dto.AdminUserListFilters) (*dto.AdminUserListResponse, error) {
	page := filters.Page
	if page <= 0 {
		page = 1
	}
	perPage := filters.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	users, total, err := s.users.ListFiltered(ctx, authrepo.UserListOptions{
		Query:    filters.Query,
		IsActive: filters.IsActive,
		Role:     filters.Role,
		Offset:   (page - 1) * perPage,
		Limit:    perPage,
	})
	if err != nil {
		return nil, err
	}

	items := make([]dto.AdminUserItem, 0, len(users))
	for i := range users {
		items = append(items, toAdminUserItem(&users[i]))
	}

	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}

	return &dto.AdminUserListResponse{
		Items: items,
		Meta: dto.AdminUserListMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *adminUserService) Get(ctx context.Context, userID uuid.UUID) (*dto.AdminUserItem, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	item := toAdminUserItem(user)
	return &item, nil
}

func (s *adminUserService) Create(ctx context.Context, actorID uuid.UUID, req *dto.AdminCreateUserRequest) (*dto.AdminUserItem, error) {
	if req == nil {
		return nil, errors.NewBadRequest("Request body is required", nil)
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		return nil, errors.NewBadRequest("Email is required", nil)
	}
	if existing, err := s.users.FindByEmail(ctx, email); err == nil && existing != nil {
		return nil, errors.NewConflict("Email is already in use", nil)
	} else if err != nil && err != errors.ErrUserNotFound {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.Split(email, "@")[0]
	}

	roles := req.Roles
	if len(roles) == 0 {
		roles = []string{string(utils.RoleUser)}
	}
	perms, err := normalizePanelPermissions(req.PanelPermissions)
	if err != nil {
		return nil, err
	}
	if !containsRole(roles, string(utils.RoleAdmin)) && len(perms) == 0 {
		return nil, errors.NewBadRequest("Select at least one panel section, or grant the admin role", nil)
	}

	points := 0
	if req.Points != nil {
		if *req.Points < 0 {
			return nil, errors.NewBadRequest("Points cannot be negative", nil)
		}
		points = *req.Points
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}

	user := &authmodels.User{
		Email:            email,
		PasswordHash:     "",
		FirstName:        name,
		LastName:         "",
		Phone:            strings.TrimSpace(req.Phone),
		Points:           points,
		IsActive:         active,
		IsVerified:       true,
		PanelPermissions: authmodels.StringList(perms),
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	roleIDs, roleErr := s.resolveRoleIDs(ctx, roles)
	if roleErr != nil {
		return nil, roleErr
	}
	if err := s.users.ReplaceRoles(ctx, user.ID, roleIDs); err != nil {
		return nil, err
	}

	logger.Info().
		Str("actor_id", actorID.String()).
		Str("user_id", user.ID.String()).
		Str("email", email).
		Msg("admin_user_created")

	return s.Get(ctx, user.ID)
}

func (s *adminUserService) Update(
	ctx context.Context,
	actorID, userID uuid.UUID,
	req *dto.AdminUpdateUserRequest,
) (*dto.AdminUserItem, error) {
	if req == nil {
		return nil, errors.NewBadRequest("Request body is required", nil)
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errors.NewBadRequest("Name cannot be empty", nil)
		}
		user.FirstName = name
		user.LastName = ""
	}

	if req.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*req.Email))
		if email == "" {
			return nil, errors.NewBadRequest("Email cannot be empty", nil)
		}
		if !strings.EqualFold(email, user.Email) {
			if existing, findErr := s.users.FindByEmail(ctx, email); findErr == nil && existing.ID != user.ID {
				return nil, errors.NewConflict("Email is already in use", nil)
			} else if findErr != nil && findErr != errors.ErrUserNotFound {
				return nil, findErr
			}
			user.Email = email
		}
	}

	if req.Phone != nil {
		user.Phone = strings.TrimSpace(*req.Phone)
	}

	if req.Points != nil {
		if *req.Points < 0 {
			return nil, errors.NewBadRequest("Points cannot be negative", nil)
		}
		user.Points = *req.Points
	}

	if req.IsActive != nil {
		if actorID == userID && !*req.IsActive {
			return nil, errors.NewBadRequest("You cannot deactivate your own account", nil)
		}
		user.IsActive = *req.IsActive
	}

	if req.IsVerified != nil {
		user.IsVerified = *req.IsVerified
	}

	if req.PanelPermissions != nil {
		perms, permErr := normalizePanelPermissions(req.PanelPermissions)
		if permErr != nil {
			return nil, permErr
		}
		willBeAdmin := containsRole(userRoleNames(user), string(utils.RoleAdmin))
		if req.Roles != nil {
			willBeAdmin = containsRole(req.Roles, string(utils.RoleAdmin))
		}
		if !willBeAdmin && len(perms) == 0 {
			return nil, errors.NewBadRequest("Select at least one panel section, or grant the admin role", nil)
		}
		user.PanelPermissions = authmodels.StringList(perms)
	}

	// Avoid GORM Save rewriting the roles association; roles are replaced separately.
	user.Roles = nil
	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}

	if req.Roles != nil {
		if actorID == userID && !containsRole(req.Roles, string(utils.RoleAdmin)) {
			return nil, errors.NewBadRequest("You cannot remove your own admin role", nil)
		}
		roleIDs, roleErr := s.resolveRoleIDs(ctx, req.Roles)
		if roleErr != nil {
			return nil, roleErr
		}
		if err := s.users.ReplaceRoles(ctx, userID, roleIDs); err != nil {
			return nil, err
		}
	}

	logger.Info().
		Str("actor_id", actorID.String()).
		Str("user_id", userID.String()).
		Msg("admin_user_updated")

	return s.Get(ctx, userID)
}

func (s *adminUserService) resolveRoleIDs(ctx context.Context, roleNames []string) ([]uuid.UUID, error) {
	seen := make(map[string]struct{}, len(roleNames))
	ids := make([]uuid.UUID, 0, len(roleNames))
	for _, raw := range roleNames {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		if _, ok := allowedAdminRoles[name]; !ok {
			return nil, errors.NewBadRequest("Invalid role: "+name, nil)
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		role, err := s.roles.FindByName(ctx, name)
		if err != nil {
			return nil, errors.NewBadRequest("Unknown role: "+name, err)
		}
		ids = append(ids, role.ID)
	}
	return ids, nil
}

func normalizePanelPermissions(raw []string) ([]string, error) {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		key := strings.ToLower(strings.TrimSpace(item))
		if key == "" {
			continue
		}
		if _, ok := allowedPanelPermissions[key]; !ok {
			return nil, errors.NewBadRequest("Invalid panel permission: "+key, nil)
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out, nil
}

func toAdminUserItem(user *authmodels.User) dto.AdminUserItem {
	roles := userRoleNames(user)
	perms := make([]string, 0, len(user.PanelPermissions))
	for _, p := range user.PanelPermissions {
		perms = append(perms, p)
	}
	item := dto.AdminUserItem{
		ID:               user.ID,
		Email:            user.Email,
		Name:             user.DisplayName(),
		Phone:            strings.TrimSpace(user.Phone),
		Points:           user.Points,
		IsActive:         user.IsActive,
		IsVerified:       user.IsVerified,
		Roles:            roles,
		PanelPermissions: perms,
		CreatedAt:        user.CreatedAt.UTC().Format(time.RFC3339),
	}
	if !user.UpdatedAt.IsZero() {
		item.UpdatedAt = user.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return item
}

func userRoleNames(user *authmodels.User) []string {
	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, role.Name)
	}
	return roles
}

func containsRole(roles []string, want string) bool {
	for _, role := range roles {
		if strings.EqualFold(strings.TrimSpace(role), want) {
			return true
		}
	}
	return false
}
