package ticket

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/neuland-ingolstadt/discord-bot/internal/config"
)

const (
	welcomeMessage = "Vielen Dank für dein Interesse an unserem Verein. Ein Vereinsmitglied wird sich in Kürze hier bei dir melden"
	topicPrefix    = "ticket-user:"
)

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

	userID := event.Member.User.ID
	exists, err := s.ticketExistsForUser(sess, userID)
	if err != nil {
		log.Printf("ticket: check existing for %s: %v", userID, err)
		return
	}
	if exists {
		log.Printf("ticket: skip %s, channel already exists", userID)
		return
	}

	channel, err := s.createTicketChannel(sess, userID)
	if err != nil {
		log.Printf("ticket: create for %s: %v", userID, err)
		return
	}

	_, err = sess.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
		Content: welcomeMessage,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Ticket schließen",
						Style:    discordgo.DangerButton,
						CustomID: CustomIDClose,
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("ticket: welcome message in %s: %v", channel.ID, err)
		return
	}

	log.Printf("ticket: opened %s for user %s", channel.Name, userID)
}

func (s *Service) createTicketChannel(sess *discordgo.Session, userID string) (*discordgo.Channel, error) {
	suffix, err := randomSuffix(4)
	if err != nil {
		return nil, err
	}

	viewSendHistory := int64(discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory)

	return sess.GuildChannelCreateComplex(s.cfg.GuildID, discordgo.GuildChannelCreateData{
		Name:     "ticket-" + suffix,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: s.cfg.OnboardingCategoryID,
		Topic:    topicPrefix + userID,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{
				ID:   s.cfg.GuildID, // @everyone
				Type: discordgo.PermissionOverwriteTypeRole,
				Deny: discordgo.PermissionViewChannel,
			},
			{
				ID:    userID,
				Type:  discordgo.PermissionOverwriteTypeMember,
				Allow: viewSendHistory,
			},
			{
				ID:    s.cfg.VorstandRoleID,
				Type:  discordgo.PermissionOverwriteTypeRole,
				Allow: viewSendHistory,
			},
			{
				ID:    s.cfg.ManagementRoleID,
				Type:  discordgo.PermissionOverwriteTypeRole,
				Allow: viewSendHistory,
			},
		},
	})
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
		if strings.HasPrefix(ch.Name, "ticket-") {
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

const suffixAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func randomSuffix(n int) (string, error) {
	var b strings.Builder
	b.Grow(n)
	max := big.NewInt(int64(len(suffixAlphabet)))
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("random suffix: %w", err)
		}
		b.WriteByte(suffixAlphabet[idx.Int64()])
	}
	return b.String(), nil
}
