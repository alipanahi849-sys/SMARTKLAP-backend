package container

import (
	"clap/internal/modules/auth/repository"
	"clap/internal/modules/auth/service"
)

type Container struct {
	UserRepository         repository.UserRepository
	RoleRepository         repository.RoleRepository
	RefreshTokenRepository repository.RefreshTokenRepository
	AuthService            service.AuthService
}

var AppContainer *Container

func Initialize() {
	userRepo := repository.NewUserRepository()
	roleRepo := repository.NewRoleRepository()
	refreshTokenRepo := repository.NewRefreshTokenRepository()

	authService := service.NewAuthService(userRepo, roleRepo, refreshTokenRepo)

	AppContainer = &Container{
		UserRepository:         userRepo,
		RoleRepository:         roleRepo,
		RefreshTokenRepository: refreshTokenRepo,
		AuthService:            authService,
	}
}

func GetContainer() *Container {
	return AppContainer
}
