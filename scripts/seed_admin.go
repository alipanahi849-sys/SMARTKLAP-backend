package main

import (
	"fmt"
	"os"
	"strings"

	"clap/internal/modules/adminauth/models"
	"clap/internal/shared/config"
	"clap/internal/shared/database"
	"clap/internal/shared/logger"
)

// Seeds (or updates) an admin_users row from ADMIN_SEED_EMAIL / ADMIN_SEED_NAME.
// Usage:
//
//	ADMIN_SEED_EMAIL=ops@example.com ADMIN_SEED_NAME="Ops" go run ./scripts/seed_admin.go
func main() {
	if err := config.LoadFromEnv(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := database.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	email := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_SEED_EMAIL")))
	name := strings.TrimSpace(os.Getenv("ADMIN_SEED_NAME"))
	if email == "" {
		fmt.Fprintln(os.Stderr, "ADMIN_SEED_EMAIL is required")
		os.Exit(1)
	}
	if name == "" {
		name = "Admin"
	}

	db := database.GetDB()
	var existing models.AdminUser
	err := db.Where("email = ?", email).First(&existing).Error
	if err == nil {
		existing.FirstName = name
		existing.IsActive = true
		if saveErr := db.Save(&existing).Error; saveErr != nil {
			fmt.Fprintln(os.Stderr, saveErr)
			os.Exit(1)
		}
		logger.Info().Str("email", email).Str("id", existing.ID.String()).Msg("admin_user_updated")
		fmt.Println(existing.ID.String())
		return
	}

	admin := &models.AdminUser{
		Email:     email,
		FirstName: name,
		IsActive:  true,
	}
	if createErr := db.Create(admin).Error; createErr != nil {
		fmt.Fprintln(os.Stderr, createErr)
		os.Exit(1)
	}
	logger.Info().Str("email", email).Str("id", admin.ID.String()).Msg("admin_user_created")
	fmt.Println(admin.ID.String())
}
