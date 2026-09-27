package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

const connectMessage = "GitHub und Discord verknüpfst du in Neuland Connect unter Konten. Melde dich dort mit deinem Vereinskonto an."

func (s *Service) handleConnect(sess *discordgo.Session, event *discordgo.InteractionCreate) {
	err := sess.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: connectMessage,
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label: "Konten öffnen",
							Style: discordgo.LinkButton,
							URL:   s.connectURL,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("commands: connect respond: %v", err)
	}
}
