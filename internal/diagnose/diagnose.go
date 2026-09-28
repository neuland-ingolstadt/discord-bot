package diagnose

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/neuland-ingolstadt/discord-bot/internal/config"
)

// Run checks guild setup and logs OK / WARN / FAIL lines.
// Returns an error only when ticket creation cannot work (hard failures).
func Run(sess *discordgo.Session, cfg *config.Config) error {
	log.Println("diagnose: starting")

	var fails, warns []string
	check := func(ok bool, fail bool, msg string) {
		if ok {
			log.Printf("diagnose: OK   %s", msg)
			return
		}
		if fail {
			fails = append(fails, msg)
			log.Printf("diagnose: FAIL %s", msg)
			return
		}
		warns = append(warns, msg)
		log.Printf("diagnose: WARN %s", msg)
	}

	if sess.State == nil || sess.State.User == nil {
		return fmt.Errorf("bot user not available in session state")
	}
	botID := sess.State.User.ID
	check(true, false, fmt.Sprintf("logged in as %s (%s)", sess.State.User.Username, botID))

	guild, err := sess.Guild(cfg.GuildID)
	if err != nil {
		check(false, true, fmt.Sprintf("guild %s: %v", cfg.GuildID, err))
		return summaryError(fails, warns)
	}
	check(true, false, fmt.Sprintf("guild %s (%s)", guild.Name, guild.ID))

	me, err := sess.GuildMember(cfg.GuildID, botID)
	if err != nil {
		check(false, true, fmt.Sprintf("bot is not a member of the guild: %v", err))
		return summaryError(fails, warns)
	}
	check(true, false, "bot is a guild member")

	roles, err := sess.GuildRoles(cfg.GuildID)
	if err != nil {
		check(false, true, fmt.Sprintf("fetch roles: %v", err))
		return summaryError(fails, warns)
	}
	roleByID := make(map[string]*discordgo.Role, len(roles))
	for _, r := range roles {
		roleByID[r.ID] = r
	}

	for _, item := range []struct {
		name string
		id   string
	}{
		{"INTERESSENT_ROLE_ID", cfg.InteressentRoleID},
		{"VORSTAND_ROLE_ID", cfg.VorstandRoleID},
		{"MANAGEMENT_ROLE_ID", cfg.ManagementRoleID},
	} {
		if r, ok := roleByID[item.id]; ok {
			check(true, false, fmt.Sprintf("%s = %s (%s)", item.name, r.Name, r.ID))
		} else {
			check(false, true, fmt.Sprintf("%s %s not found in guild", item.name, item.id))
		}
	}

	category, err := sess.Channel(cfg.OnboardingCategoryID)
	if err != nil {
		check(false, true, fmt.Sprintf("ONBOARDING_CATEGORY_ID %s: %v", cfg.OnboardingCategoryID, err))
		return summaryError(fails, warns)
	}
	if category.GuildID != cfg.GuildID {
		check(false, true, fmt.Sprintf("category %s belongs to guild %s, not %s", category.ID, category.GuildID, cfg.GuildID))
	} else if category.Type != discordgo.ChannelTypeGuildCategory {
		check(false, true, fmt.Sprintf("%s is type %d, expected category", category.Name, category.Type))
	} else {
		check(true, false, fmt.Sprintf("onboarding category %s (%s)", category.Name, category.ID))
	}

	// Ensure guild.Roles is populated for permission calculation.
	if len(guild.Roles) == 0 {
		guild.Roles = roles
	}

	perms := memberPermissions(guild, category, botID, me.Roles)
	for _, need := range []struct {
		bit  int64
		name string
	}{
		{discordgo.PermissionViewChannel, "View Channel"},
		{discordgo.PermissionManageChannels, "Manage Channels"},
		{discordgo.PermissionManageRoles, "Manage Roles"},
		{discordgo.PermissionSendMessages, "Send Messages"},
		{discordgo.PermissionReadMessageHistory, "Read Message History"},
	} {
		has := perms&need.bit == need.bit || perms&discordgo.PermissionAdministrator == discordgo.PermissionAdministrator
		check(has, true, fmt.Sprintf("bot permission on category: %s", need.name))
	}

	// Staff access for tickets comes from category overwrites (channels sync, then only the member is added).
	checkRoleView := func(roleID, label string) {
		r := roleByID[roleID]
		name := roleID
		if r != nil {
			name = r.Name
		}
		allow, deny := roleOverwrite(category, roleID)
		if allow&discordgo.PermissionViewChannel != 0 {
			check(true, false, fmt.Sprintf("category grants View Channel to %s (%s)", label, name))
			return
		}
		if deny&discordgo.PermissionViewChannel != 0 {
			check(false, false, fmt.Sprintf("category denies View Channel for %s (%s) — staff will not see tickets", label, name))
			return
		}
		check(false, false, fmt.Sprintf("category has no View Channel overwrite for %s (%s) — add it so staff can see tickets", label, name))
	}
	checkRoleView(cfg.VorstandRoleID, "Vorstand")
	checkRoleView(cfg.ManagementRoleID, "Management")

	everyoneAllow, everyoneDeny := roleOverwrite(category, cfg.GuildID)
	if everyoneDeny&discordgo.PermissionViewChannel != 0 {
		check(true, false, "category denies View Channel for @everyone")
	} else if everyoneAllow&discordgo.PermissionViewChannel != 0 {
		check(false, false, "category allows @everyone View Channel — tickets may be visible to everyone until synced permissions are correct")
	} else {
		check(false, false, "category does not deny @everyone View Channel — recommended for private tickets")
	}

	if len(fails) == 0 && len(warns) == 0 {
		log.Println("diagnose: all checks passed")
		return nil
	}
	if len(fails) == 0 {
		log.Printf("diagnose: %d warning(s), bot will start", len(warns))
		return nil
	}
	return summaryError(fails, warns)
}

func summaryError(fails, warns []string) error {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%d hard failure(s)", len(fails)))
	if len(warns) > 0 {
		b.WriteString(fmt.Sprintf(", %d warning(s)", len(warns)))
	}
	b.WriteString(" — see diagnose log lines above")
	return fmt.Errorf("%s", b.String())
}

func roleOverwrite(ch *discordgo.Channel, roleID string) (allow, deny int64) {
	for _, ow := range ch.PermissionOverwrites {
		if ow.Type == discordgo.PermissionOverwriteTypeRole && ow.ID == roleID {
			return ow.Allow, ow.Deny
		}
	}
	return 0, 0
}

// memberPermissions mirrors discordgo's calculation (unexported there).
func memberPermissions(guild *discordgo.Guild, channel *discordgo.Channel, userID string, roles []string) (apermissions int64) {
	if userID == guild.OwnerID {
		return discordgo.PermissionAll
	}

	for _, role := range guild.Roles {
		if role.ID == guild.ID {
			apermissions |= role.Permissions
			break
		}
	}
	for _, role := range guild.Roles {
		for _, roleID := range roles {
			if role.ID == roleID {
				apermissions |= role.Permissions
				break
			}
		}
	}
	if apermissions&discordgo.PermissionAdministrator == discordgo.PermissionAdministrator {
		return discordgo.PermissionAll
	}

	for _, overwrite := range channel.PermissionOverwrites {
		if guild.ID == overwrite.ID {
			apermissions &= ^overwrite.Deny
			apermissions |= overwrite.Allow
			break
		}
	}

	var denies, allows int64
	for _, overwrite := range channel.PermissionOverwrites {
		for _, roleID := range roles {
			if overwrite.Type == discordgo.PermissionOverwriteTypeRole && roleID == overwrite.ID {
				denies |= overwrite.Deny
				allows |= overwrite.Allow
				break
			}
		}
	}
	apermissions &= ^denies
	apermissions |= allows

	for _, overwrite := range channel.PermissionOverwrites {
		if overwrite.Type == discordgo.PermissionOverwriteTypeMember && overwrite.ID == userID {
			apermissions &= ^overwrite.Deny
			apermissions |= overwrite.Allow
			break
		}
	}
	return apermissions
}
