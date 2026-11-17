package discord

import (
	"context"
	"fmt"
	"unicode/utf16"

	"github.com/bwmarrin/discordgo"
)

const maxMessageUnits = 2000

func (b *Bot) replyToMessage(ctx context.Context, m *discordgo.Message, answer string) error {
	ctx, cancel := context.WithTimeout(ctx, b.config.ReplyTimeout)
	defer cancel()

	for i, chunk := range splitMessage(answer) {
		out := &discordgo.MessageSend{
			Content: chunk,
			AllowedMentions: &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{},
			},
		}

		if i == 0 {
			out.Reference = &discordgo.MessageReference{
				MessageID:       m.ID,
				ChannelID:       m.ChannelID,
				GuildID:         m.GuildID,
				FailIfNotExists: new(false),
			}
		}

		if _, err := b.sess.ChannelMessageSendComplex(m.ChannelID, out, discordgo.WithContext(ctx)); err != nil {
			return fmt.Errorf("%w: %w", ErrSendReply, err)
		}
	}

	return nil
}

func (b *Bot) respondToCommand(
	ctx context.Context,
	source *discordgo.Interaction,
	answer string,
	flags discordgo.MessageFlags,
) error {
	ctx, cancel := context.WithTimeout(ctx, b.config.ReplyTimeout)
	defer cancel()

	if answer == "" {
		return b.sess.InteractionResponseDelete(source, discordgo.WithContext(ctx))
	}

	for i, chunk := range splitMessage(answer) {
		if i == 0 {
			if _, err := b.sess.InteractionResponseEdit(source, &discordgo.WebhookEdit{
				Content: new(chunk),
				AllowedMentions: &discordgo.MessageAllowedMentions{
					Parse: []discordgo.AllowedMentionType{},
				},
			}, discordgo.WithContext(ctx)); err != nil {
				return err
			}
			continue
		}

		if _, err := b.sess.FollowupMessageCreate(source, true, &discordgo.WebhookParams{
			Content: chunk,
			Flags:   flags,
			AllowedMentions: &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{},
			},
		}, discordgo.WithContext(ctx)); err != nil {
			return err
		}
	}

	return nil
}

func splitMessage(text string) []string {
	var chunks []string
	start, units := 0, 0

	for index, character := range text {
		size := utf16.RuneLen(character)

		if units+size > maxMessageUnits {
			chunks = append(chunks, text[start:index])
			start, units = index, 0
		}

		units += size
	}

	if start < len(text) {
		chunks = append(chunks, text[start:])
	}

	return chunks
}
