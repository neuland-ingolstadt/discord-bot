package ticket

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// HandleMemberRemove notifies staff when an onboarding user leaves the guild mid-ticket.
func (s *Service) HandleMemberRemove(sess *discordgo.Session, event *discordgo.GuildMemberRemove) {
	if event.GuildID != s.cfg.GuildID {
		return
	}
	if event.User == nil {
		return
	}

	user := event.User
	who := userLabel(user)
	channel, err := s.findTicketChannelForUser(sess, user.ID)
	if err != nil {
		log.Printf("ticket: find channel after leave for %s: %v", who, err)
		return
	}
	if channel == nil {
		return
	}

	if _, err := sess.ChannelMessageSendComplex(channel.ID, s.memberLeftMessage(user)); err != nil {
		log.Printf("ticket: leave message in %s for %s: %v", channel.Name, who, err)
		return
	}
	log.Printf("ticket: %s left guild; notified in %s", who, channel.Name)
}

func (s *Service) memberLeftMessage(user *discordgo.User) *discordgo.MessageSend {
	accent := welcomeAccentColor
	return &discordgo.MessageSend{
		Flags: discordgo.MessageFlagsIsComponentsV2,
		Components: []discordgo.MessageComponent{
			discordgo.Container{
				AccentColor: &accent,
				Components: []discordgo.MessageComponent{
					discordgo.TextDisplay{
						Content: "### Mitglied hat den Server verlassen",
					},
					discordgo.TextDisplay{
						Content: fmt.Sprintf(
							"**%s** (`%s`) ist nicht mehr auf dem Server. Das Onboarding-Ticket kann geschlossen werden.",
							userLabel(user),
							user.ID,
						),
					},
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
			},
		},
	}
}
