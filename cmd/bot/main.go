package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/neuland-ingolstadt/discord-bot/internal/commands"
	"github.com/neuland-ingolstadt/discord-bot/internal/config"
	"github.com/neuland-ingolstadt/discord-bot/internal/ticket"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		log.Fatalf("discord session: %v", err)
	}

	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers
	session.LogLevel = discordgo.LogWarning

	tickets := ticket.New(cfg)
	cmds := commands.New(cfg.ConnectURL, cfg.VorstandRoleID, cfg.ManagementRoleID)
	session.AddHandler(tickets.HandleMemberUpdate)
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		cmds.HandleInteraction(s, i)
		tickets.HandleInteraction(s, i)
	})

	if err := session.Open(); err != nil {
		log.Fatalf("discord open: %v", err)
	}
	defer session.Close()

	if err := cmds.Register(session, cfg.GuildID); err != nil {
		log.Fatalf("commands register: %v", err)
	}

	if err := session.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status: "online",
		Activities: []*discordgo.Activity{
			{
				Name: "Onboarding · /connect",
				Type: discordgo.ActivityTypeWatching,
			},
		},
	}); err != nil {
		log.Printf("presence: %v", err)
	}

	log.Printf("bot online as %s (%s %s)", session.State.User.Username, version, commit)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down")
}
