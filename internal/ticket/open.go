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
	who := userLabel(user)
	exists, err := s.ticketExistsForUser(sess, user.ID)
	if err != nil {
		log.Printf("ticket: check existing for %s: %v", who, err)
		return
	}
	if exists {
		log.Printf("ticket: skip %s, channel already exists", who)
		return
	}

	channel, err := s.createTicketChannel(sess, user)
	if err != nil {
		log.Printf("ticket: create for %s: %v", who, err)
		return
	}

	if _, err := sess.ChannelMessageSendComplex(channel.ID, s.welcomeMessage(sess, user)); err != nil {
		log.Printf("ticket: welcome message in %s: %v", channel.Name, err)
		return
	}

	log.Printf("ticket: opened %s for %s", channel.Name, who)
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

// Neuland accent green (from brand).
const welcomeAccentColor = 0x92DBA5

const (
	websiteURL = "https://neuland-ingolstadt.de/de"
	joinURL    = "https://join.neuland-ingolstadt.de/"
)

func (s *Service) welcomeMessage(sess *discordgo.Session, user *discordgo.User) *discordgo.MessageSend {
	divider := true
	spacing := discordgo.SeparatorSpacingSizeLarge
	accent := welcomeAccentColor

	var body []discordgo.MessageComponent
	if icon := guildIconURL(sess, s.cfg.GuildID); icon != "" {
		desc := "Neuland"
		body = append(body, discordgo.Section{
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{
					Content: fmt.Sprintf("## Hey <@%s>", user.ID),
				},
				discordgo.TextDisplay{
					Content: "Danke für dein Interesse an unserem Verein!",
				},
			},
			Accessory: discordgo.Thumbnail{
				Media:       discordgo.UnfurledMediaItem{URL: icon},
				Description: &desc,
			},
		})
	} else {
		body = append(body,
			discordgo.TextDisplay{
				Content: fmt.Sprintf("## Hey <@%s>", user.ID),
			},
			discordgo.TextDisplay{
				Content: "Danke für dein Interesse an unserem Verein!",
			},
		)
	}

	body = append(body,
		discordgo.Separator{
			Divider: &divider,
			Spacing: &spacing,
		},
		discordgo.TextDisplay{
			Content: "### Was passiert als Nächstes?\n" +
				"1. Ein Vereinsmitglied meldet sich hier bei dir.\n" +
				"2. Ihr lernt euch und deine Interessen kennen und schaut, wie du dich einbringen kannst.\n" +
				"3. Stell gerne alle Fragen, die du an uns hast.",
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label: "Website",
					Style: discordgo.LinkButton,
					URL:   websiteURL,
				},
				discordgo.Button{
					Label: "Beitrittsformular",
					Style: discordgo.LinkButton,
					URL:   joinURL,
				},
			},
		},
	)

	return &discordgo.MessageSend{
		Flags: discordgo.MessageFlagsIsComponentsV2,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Users: []string{user.ID},
		},
		Components: []discordgo.MessageComponent{
			discordgo.Container{
				AccentColor: &accent,
				Components:  body,
			},
		},
	}
}

func guildIconURL(sess *discordgo.Session, guildID string) string {
	guild, err := sess.State.Guild(guildID)
	if err != nil || guild == nil || guild.Icon == "" {
		guild, err = sess.Guild(guildID)
		if err != nil || guild == nil || guild.Icon == "" {
			return ""
		}
	}
	ext := "png"
	if strings.HasPrefix(guild.Icon, "a_") {
		ext = "gif"
	}
	return fmt.Sprintf("https://cdn.discordapp.com/icons/%s/%s.%s?size=256", guild.ID, guild.Icon, ext)
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
	ch, err := s.findTicketChannelForUser(sess, userID)
	if err != nil {
		return false, err
	}
	return ch != nil, nil
}

func (s *Service) findTicketChannelForUser(sess *discordgo.Session, userID string) (*discordgo.Channel, error) {
	channels, err := sess.GuildChannels(s.cfg.GuildID)
	if err != nil {
		return nil, err
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
			return ch, nil
		}
		if strings.HasPrefix(ch.Name, "welcome-") || strings.HasPrefix(ch.Name, "ticket-") {
			for _, ow := range ch.PermissionOverwrites {
				if ow.Type == discordgo.PermissionOverwriteTypeMember && ow.ID == userID {
					return ch, nil
				}
			}
		}
	}
	return nil, nil
}

func hasRole(roles []string, roleID string) bool {
	for _, id := range roles {
		if id == roleID {
			return true
		}
	}
	return false
}

func userLabel(user *discordgo.User) string {
	if user == nil {
		return "unknown"
	}
	if user.Username != "" {
		return user.Username
	}
	if user.GlobalName != "" {
		return user.GlobalName
	}
	return user.ID
}
