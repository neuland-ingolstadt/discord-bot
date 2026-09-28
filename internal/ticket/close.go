package ticket

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	CommandName          = "ticket"
	SubcommandClose      = "close"
	CustomIDClose        = "ticket_close"
	CustomIDCloseConfirm = "ticket_close_confirm"
	CustomIDCloseCancel  = "ticket_close_cancel"
)

// HandleInteraction routes ticket slash commands and button interactions.
func (s *Service) HandleInteraction(sess *discordgo.Session, event *discordgo.InteractionCreate) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ticket: panic handling interaction: %v", r)
		}
	}()

	if event == nil || event.Interaction == nil {
		log.Printf("ticket: interaction event missing interaction payload")
		return
	}

	switch event.Type {
	case discordgo.InteractionApplicationCommand:
		s.handleSlashCommand(sess, event)
	case discordgo.InteractionMessageComponent:
		s.handleComponent(sess, event)
	}
}

func (s *Service) handleSlashCommand(sess *discordgo.Session, event *discordgo.InteractionCreate) {
	data := event.ApplicationCommandData()
	if data.Name != CommandName {
		return
	}

	guildID := event.GuildID
	if guildID == "" && event.ChannelID != "" {
		if ch, err := sess.Channel(event.ChannelID); err == nil {
			guildID = ch.GuildID
		}
	}
	if guildID != "" && guildID != s.cfg.GuildID {
		log.Printf("ticket: ignore slash guild=%s want=%s", guildID, s.cfg.GuildID)
		return
	}

	if len(data.Options) == 0 {
		_ = respondEphemeral(sess, event, "Unbekannte Ticket-Aktion.")
		return
	}

	switch data.Options[0].Name {
	case SubcommandClose:
		s.handleCloseSlash(sess, event)
	default:
		_ = respondEphemeral(sess, event, "Unbekannte Ticket-Aktion.")
	}
}

func (s *Service) handleComponent(sess *discordgo.Session, event *discordgo.InteractionCreate) {
	customID := ""
	if data, ok := event.Data.(discordgo.MessageComponentInteractionData); ok {
		customID = data.CustomID
	} else {
		log.Printf("ticket: component interaction without component data (type=%v)", event.Type)
		_ = respondEphemeral(sess, event, "Interaktion konnte nicht gelesen werden.")
		return
	}

	if !strings.HasPrefix(customID, "ticket_") {
		return
	}

	guildID := event.GuildID
	if guildID == "" && event.ChannelID != "" {
		if ch, err := sess.Channel(event.ChannelID); err == nil {
			guildID = ch.GuildID
		}
	}
	if guildID != "" && guildID != s.cfg.GuildID {
		log.Printf("ticket: ignore interaction guild=%s want=%s custom_id=%s", guildID, s.cfg.GuildID, customID)
		return
	}

	log.Printf("ticket: interaction custom_id=%s user=%s channel=%s", customID, memberUserLabel(event), event.ChannelID)

	switch customID {
	case CustomIDClose:
		// Kept for older welcome messages that still have the button.
		s.handleCloseRequest(sess, event)
	case CustomIDCloseConfirm:
		s.handleCloseConfirm(sess, event)
	case CustomIDCloseCancel:
		s.handleCloseCancel(sess, event)
	default:
		log.Printf("ticket: unknown custom_id=%s", customID)
		_ = respondEphemeral(sess, event, "Unbekannte Ticket-Aktion.")
	}
}

func (s *Service) handleCloseSlash(sess *discordgo.Session, event *discordgo.InteractionCreate) {
	if !s.isStaff(event) {
		if err := respondEphemeral(sess, event, "Nur Vorstand und Management können Tickets schließen."); err != nil {
			log.Printf("ticket: deny close slash: %v", err)
		}
		return
	}

	ok, err := s.isTicketChannel(sess, event.ChannelID)
	if err != nil {
		log.Printf("ticket: check channel %s: %v", event.ChannelID, err)
		_ = respondEphemeral(sess, event, "Kanal konnte nicht geprüft werden.")
		return
	}
	if !ok {
		_ = respondEphemeral(sess, event, "Dieser Befehl funktioniert nur in einem Onboarding-Ticket.")
		return
	}

	s.promptCloseConfirm(sess, event)
}

func (s *Service) handleCloseRequest(sess *discordgo.Session, event *discordgo.InteractionCreate) {
	if !s.isStaff(event) {
		if err := respondEphemeral(sess, event, "Nur Vorstand und Management können Tickets schließen."); err != nil {
			log.Printf("ticket: deny close respond: %v", err)
		}
		return
	}

	s.promptCloseConfirm(sess, event)
}

func (s *Service) promptCloseConfirm(sess *discordgo.Session, event *discordgo.InteractionCreate) {
	// ACK immediately (Discord requires a response within 3s), then attach confirm buttons.
	err := sess.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("ticket: close defer: %v", err)
		return
	}

	content := "Ticket wirklich schließen?"
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Bestätigen",
					Style:    discordgo.DangerButton,
					CustomID: CustomIDCloseConfirm,
				},
				discordgo.Button{
					Label:    "Abbrechen",
					Style:    discordgo.SecondaryButton,
					CustomID: CustomIDCloseCancel,
				},
			},
		},
	}
	_, err = sess.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
		Content:    &content,
		Components: &components,
	})
	if err != nil {
		log.Printf("ticket: close prompt edit: %v", err)
	}
}

func (s *Service) handleCloseConfirm(sess *discordgo.Session, event *discordgo.InteractionCreate) {
	if !s.isStaff(event) {
		if err := respondEphemeral(sess, event, "Nur Vorstand und Management können Tickets schließen."); err != nil {
			log.Printf("ticket: deny confirm respond: %v", err)
		}
		return
	}

	channelID := event.ChannelID
	err := sess.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Ticket wird geschlossen…",
			Components: []discordgo.MessageComponent{},
		},
	})
	if err != nil {
		log.Printf("ticket: close confirm respond: %v", err)
		return
	}

	if _, err := sess.ChannelDelete(channelID); err != nil {
		log.Printf("ticket: delete channel %s: %v", channelID, err)
		_, _ = sess.FollowupMessageCreate(event.Interaction, true, &discordgo.WebhookParams{
			Content: "Kanal konnte nicht gelöscht werden. Bitte manuell prüfen.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}
	log.Printf("ticket: closed/deleted channel %s by %s", channelID, memberUserLabel(event))
}

func (s *Service) handleCloseCancel(sess *discordgo.Session, event *discordgo.InteractionCreate) {
	err := sess.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Abgebrochen",
			Components: []discordgo.MessageComponent{},
		},
	})
	if err != nil {
		log.Printf("ticket: close cancel: %v", err)
	}
}

func (s *Service) isTicketChannel(sess *discordgo.Session, channelID string) (bool, error) {
	ch, err := sess.Channel(channelID)
	if err != nil {
		return false, err
	}
	if ch.GuildID != s.cfg.GuildID {
		return false, nil
	}
	if ch.Type != discordgo.ChannelTypeGuildText {
		return false, nil
	}
	if ch.ParentID != s.cfg.OnboardingCategoryID {
		return false, nil
	}
	if strings.HasPrefix(ch.Topic, topicPrefix) {
		return true, nil
	}
	return strings.HasPrefix(ch.Name, "welcome-") || strings.HasPrefix(ch.Name, "ticket-"), nil
}

func (s *Service) isStaff(event *discordgo.InteractionCreate) bool {
	if event.Member == nil {
		log.Printf("ticket: staff check failed: member is nil (user=%s)", memberUserLabel(event))
		return false
	}
	ok := s.cfg.HasStaffRole(event.Member.Roles)
	if !ok {
		log.Printf("ticket: staff check failed for %s roles=%v", memberUserLabel(event), event.Member.Roles)
	}
	return ok
}

func memberUserLabel(event *discordgo.InteractionCreate) string {
	if event.Member != nil && event.Member.User != nil {
		return userLabel(event.Member.User)
	}
	if event.User != nil {
		return userLabel(event.User)
	}
	return "unknown"
}

func respondEphemeral(sess *discordgo.Session, event *discordgo.InteractionCreate, content string) error {
	return sess.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
