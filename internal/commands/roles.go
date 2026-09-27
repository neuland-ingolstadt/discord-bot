package commands

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const discordMessageLimit = 2000

func (s *Service) handleRoles(sess *discordgo.Session, event *discordgo.InteractionCreate) {
	if !s.isStaff(event) {
		if err := respondEphemeral(sess, event, "Nur Vorstand und Management können diesen Befehl nutzen."); err != nil {
			log.Printf("commands: roles deny: %v", err)
		}
		return
	}

	guildID := event.GuildID
	if guildID == "" {
		_ = respondEphemeral(sess, event, "Dieser Befehl funktioniert nur auf einem Server.")
		return
	}

	roles, err := sess.GuildRoles(guildID)
	if err != nil {
		log.Printf("commands: guild roles: %v", err)
		_ = respondEphemeral(sess, event, "Rollen konnten nicht geladen werden.")
		return
	}

	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Position > roles[j].Position
	})

	var b strings.Builder
	b.WriteString("```\n")
	for _, role := range roles {
		fmt.Fprintf(&b, "`%s` — `%s`\n", escapeInlineCode(role.Name), role.ID)
	}
	b.WriteString("```")
	body := b.String()
	if len(roles) == 0 {
		body = "_Keine Rollen gefunden._"
	}

	chunks := chunkCodeBlocks(body, discordMessageLimit)
	err = sess.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: chunks[0],
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("commands: roles respond: %v", err)
		return
	}

	for _, chunk := range chunks[1:] {
		_, err := sess.FollowupMessageCreate(event.Interaction, true, &discordgo.WebhookParams{
			Content: chunk,
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("commands: roles followup: %v", err)
			return
		}
	}
}

func escapeInlineCode(s string) string {
	return strings.ReplaceAll(s, "`", "'")
}

// chunkCodeBlocks splits a fenced code block into multiple valid fenced messages.
func chunkCodeBlocks(s string, limit int) []string {
	if len(s) <= limit {
		return []string{s}
	}

	inner := strings.TrimSuffix(strings.TrimPrefix(s, "```\n"), "\n```")
	lines := strings.Split(inner, "\n")

	var chunks []string
	var b strings.Builder
	b.WriteString("```\n")
	for _, line := range lines {
		// ``` + line + newline + closing ```
		if b.Len()+len(line)+5 > limit && b.Len() > 4 {
			b.WriteString("```")
			chunks = append(chunks, b.String())
			b.Reset()
			b.WriteString("```\n")
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString("```")
	chunks = append(chunks, b.String())
	return chunks
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
