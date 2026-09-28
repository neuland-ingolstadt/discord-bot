package ticket

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/neuland-ingolstadt/discord-bot/internal/config"
)

const topicPrefix = "ticket-user:"

var nonChannelChars = regexp.MustCompile(`[^a-z0-9-]+`)

// Service handles onboarding ticket lifecycle.
type Service struct {
	cfg *config.Config
}

// New creates a ticket Service.
func New(cfg *config.Config) *Service {
	return &Service{cfg: cfg}
}

// HandleMemberUpdate opens a ticket when the Interessent role is newly assigned.
func (s *Service) HandleMemberUpdate(sess *discordgo.Session, event *discordgo.GuildMemberUpdate) {
	if event.GuildID != s.cfg.GuildID {
		return
	}
	if event.Member == nil || event.Member.User == nil {
		return
	}

	hadRole := false
	if event.BeforeUpdate != nil {
		hadRole = hasRole(event.BeforeUpdate.Roles, s.cfg.InteressentRoleID)
	}
	hasRoleNow := hasRole(event.Member.Roles, s.cfg.InteressentRoleID)
	if hadRole || !hasRoleNow {
		return
	}

	user := event.Member.User
	userID := user.ID
	exists, err := s.ticketExistsForUser(sess, userID)
	if err != nil {
		log.Printf("ticket: check existing for %s: %v", userID, err)
		return
	}
	if exists {
		log.Printf("ticket: skip %s, channel already exists", userID)
		return
	}

	channel, err := s.createTicketChannel(sess, user)
	if err != nil {
		log.Printf("ticket: create for %s: %v", userID, err)
		return
	}

	_, err = sess.ChannelMessageSend(channel.ID, welcomeMessage(user))
	if err != nil {
		log.Printf("ticket: welcome message in %s: %v", channel.ID, err)
		return
	}

	log.Printf("ticket: opened %s for user %s", channel.Name, userID)
}

func (s *Service) createTicketChannel(sess *discordgo.Session, user *discordgo.User) (*discordgo.Channel, error) {
	// Create synced under the category so Vorstand/Management (and @everyone deny)
	// come from category overwrites — no need for the bot role to sit above staff.
	channel, err := sess.GuildChannelCreateComplex(s.cfg.GuildID, discordgo.GuildChannelCreateData{
		Name:     ticketChannelName(user.Username),
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: s.cfg.OnboardingCategoryID,
		Topic:    topicPrefix + user.ID,
	})
	if err != nil {
		return nil, err
	}

	viewSendHistory := int64(discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory)
	if err := sess.ChannelPermissionSet(channel.ID, user.ID, discordgo.PermissionOverwriteTypeMember, viewSendHistory, 0); err != nil {
		if _, delErr := sess.ChannelDelete(channel.ID); delErr != nil {
			log.Printf("ticket: rollback delete %s after permission failure: %v", channel.ID, delErr)
		}
		return nil, fmt.Errorf("member overwrite: %w", err)
	}
	return channel, nil
}

func welcomeMessage(user *discordgo.User) string {
	return fmt.Sprintf(
		"Hey <@%s> 👋\n\n"+
			"Danke für dein Interesse an unserem Verein!\n\n"+
			"Ein Vereinsmitglied meldet sich in Kürze persönlich bei dir hier im Chat. Dabei geht es darum, dich und deine Interessen kennenzulernen und gemeinsam zu schauen, wie du dich bei uns einbringen kannst.\n\n"+
			"Und natürlich kannst du die Gelegenheit auch nutzen, um alle Fragen loszuwerden, die du an uns hast!",
		user.ID,
	)
}

func ticketChannelName(username string) string {
	name := strings.ToLower(username)
	name = strings.ReplaceAll(name, "_", "-")
	name = nonChannelChars.ReplaceAllString(name, "")
	name = strings.Trim(name, "-")
	if name == "" {
		name = "user"
	}
	const prefix = "welcome-"
	const maxLen = 100 - len(prefix)
	if len(name) > maxLen {
		name = name[:maxLen]
	}
	return prefix + name
}

func (s *Service) ticketExistsForUser(sess *discordgo.Session, userID string) (bool, error) {
	channels, err := sess.GuildChannels(s.cfg.GuildID)
	if err != nil {
		return false, err
	}

	wantTopic := topicPrefix + userID
	for _, ch := range channels {
		if ch.ParentID != s.cfg.OnboardingCategoryID {
			continue
		}
		if ch.Type != discordgo.ChannelTypeGuildText {
			continue
		}
		if ch.Topic == wantTopic {
			return true, nil
		}
		if strings.HasPrefix(ch.Name, "welcome-") || strings.HasPrefix(ch.Name, "ticket-") {
			for _, ow := range ch.PermissionOverwrites {
				if ow.Type == discordgo.PermissionOverwriteTypeMember && ow.ID == userID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func hasRole(roles []string, roleID string) bool {
	for _, id := range roles {
		if id == roleID {
			return true
		}
	}
	return false
}
