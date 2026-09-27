package ticket

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

const (
	CustomIDClose        = "ticket_close"
	CustomIDCloseConfirm = "ticket_close_confirm"
	CustomIDCloseCancel  = "ticket_close_cancel"
)

// HandleInteraction routes ticket button interactions.
func (s *Service) HandleInteraction(sess *discordgo.Session, event *discordgo.InteractionCreate) {
	if event.Type != discordgo.InteractionMessageComponent {
		return
	}
	if event.GuildID != s.cfg.GuildID {
		return
	}

	switch event.MessageComponentData().CustomID {
	case CustomIDClose:
		s.handleCloseRequest(sess, event)
	case CustomIDCloseConfirm:
		s.handleCloseConfirm(sess, event)
	case CustomIDCloseCancel:
		s.handleCloseCancel(sess, event)
	}
}

func (s *Service) handleCloseRequest(sess *discordgo.Session, event *discordgo.InteractionCreate) {
	if !s.isStaff(event) {
		_ = respondEphemeral(sess, event, "Nur Vorstand und Management können Tickets schließen.")
		return
	}

	err := sess.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Ticket wirklich schließen?",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
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
			},
		},
	})
	if err != nil {
		log.Printf("ticket: close prompt: %v", err)
	}
}

func (s *Service) handleCloseConfirm(sess *discordgo.Session, event *discordgo.InteractionCreate) {
	if !s.isStaff(event) {
		_ = respondEphemeral(sess, event, "Nur Vorstand und Management können Tickets schließen.")
		return
	}

	channelID := event.ChannelID
	_ = sess.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Ticket wird geschlossen…",
			Components: []discordgo.MessageComponent{},
		},
	})

	if _, err := sess.ChannelDelete(channelID); err != nil {
		log.Printf("ticket: delete channel %s: %v", channelID, err)
		_, _ = sess.FollowupMessageCreate(event.Interaction, true, &discordgo.WebhookParams{
			Content: "Kanal konnte nicht gelöscht werden. Bitte manuell prüfen.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}
	log.Printf("ticket: closed/deleted channel %s by %s", channelID, memberUserID(event))
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

func (s *Service) isStaff(event *discordgo.InteractionCreate) bool {
	if event.Member == nil {
		return false
	}
	return s.cfg.HasStaffRole(event.Member.Roles)
}

func memberUserID(event *discordgo.InteractionCreate) string {
	if event.Member != nil && event.Member.User != nil {
		return event.Member.User.ID
	}
	if event.User != nil {
		return event.User.ID
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
