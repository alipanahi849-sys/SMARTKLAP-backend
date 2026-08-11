package main

import (
	"fmt"
	"os"

	"clap/internal/shared/config"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

func main() {
	if err := config.LoadFromEnv(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	uid := uuid.MustParse("c1000000-0000-4000-8000-000000000001")
	tok, _, err := utils.GenerateAccessToken(uid, "admin@clap.test", []string{"admin"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(tok)
}
