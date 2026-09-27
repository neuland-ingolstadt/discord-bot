package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

const (
	rolesCommandName   = "roles"
	connectCommandName = "connect"
)

// Service handles slash command registration and responses.
type Service struct {
	connectURL string
}

// New creates a slash-command Service.
func New(connectURL string) *Service {
	return &Service{connectURL: connectURL}
}

func (s *Service) definitions() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        rolesCommandName,
			Description: "List all roles in this server with their IDs",
		},
		{
			Name:        connectCommandName,
			Description: "Öffnet Neuland Connect (Konten) zum Verknüpfen von GitHub und Discord",
		},
	}
}

// Register creates or updates guild slash commands for the bot.
func (s *Service) Register(sess *discordgo.Session, guildID string) error {
	cmds, err := sess.ApplicationCommandBulkOverwrite(sess.State.User.ID, guildID, s.definitions())
	if err != nil {
		return err
	}
	for _, cmd := range cmds {
		log.Printf("commands: registered /%s", cmd.Name)
	}
	return nil
}

// HandleInteraction responds to slash commands owned by this package.
func (s *Service) HandleInteraction(sess *discordgo.Session, event *discordgo.InteractionCreate) {
	if event.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch event.ApplicationCommandData().Name {
	case rolesCommandName:
		handleRoles(sess, event)
	case connectCommandName:
		s.handleConnect(sess, event)
	}
}
