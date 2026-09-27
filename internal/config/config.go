package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds runtime settings loaded from the environment.
type Config struct {
	Token                string
	GuildID              string
	OnboardingCategoryID string
	InteressentRoleID    string
	VorstandRoleID       string
	ManagementRoleID     string
	ConnectURL           string
}

// Load reads configuration from environment variables.
// Optionally loads a local .env file when present.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Token:                strings.TrimSpace(os.Getenv("DISCORD_TOKEN")),
		GuildID:              strings.TrimSpace(os.Getenv("GUILD_ID")),
		OnboardingCategoryID: strings.TrimSpace(os.Getenv("ONBOARDING_CATEGORY_ID")),
		InteressentRoleID:    strings.TrimSpace(os.Getenv("INTERESSENT_ROLE_ID")),
		VorstandRoleID:       strings.TrimSpace(os.Getenv("VORSTAND_ROLE_ID")),
		ManagementRoleID:     strings.TrimSpace(os.Getenv("MANAGEMENT_ROLE_ID")),
		ConnectURL:           strings.TrimSpace(os.Getenv("CONNECT_URL")),
	}

	if cfg.ConnectURL == "" {
		cfg.ConnectURL = "https://connect.neuland.ing/connect"
	}

	required := map[string]string{
		"DISCORD_TOKEN":          cfg.Token,
		"GUILD_ID":               cfg.GuildID,
		"ONBOARDING_CATEGORY_ID": cfg.OnboardingCategoryID,
		"INTERESSENT_ROLE_ID":    cfg.InteressentRoleID,
		"VORSTAND_ROLE_ID":       cfg.VorstandRoleID,
		"MANAGEMENT_ROLE_ID":     cfg.ManagementRoleID,
	}
	var missing []string
	for name, value := range required {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

// IsStaffRole reports whether roleID is Vorstand or Management.
func (c *Config) IsStaffRole(roleID string) bool {
	return roleID == c.VorstandRoleID || roleID == c.ManagementRoleID
}

// HasStaffRole reports whether any of the given role IDs is staff.
func (c *Config) HasStaffRole(roleIDs []string) bool {
	for _, id := range roleIDs {
		if c.IsStaffRole(id) {
			return true
		}
	}
	return false
}
