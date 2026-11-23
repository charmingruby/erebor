package discord

import (
	"context"
	"fmt"
	"unicode/utf16"

	"github.com/bwmarrin/discordgo"

	"github.com/charmingruby/erebor/discord/api"
)

const (
	maxMessageUnits  = 2000
	maxMessageEmbeds = 10
)

func (b *Bot) sendMessageReply(ctx context.Context, m *discordgo.Message, answer string) error {
	ctx, cancel := context.WithTimeout(ctx, b.config.replyTimeout)
	defer cancel()

	for i, chunk := range splitMessage(answer) {
		out := &discordgo.MessageSend{
			Content:         chunk,
			AllowedMentions: noMentions(),
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

func (b *Bot) sendCommandReply(
	ctx context.Context,
	source *discordgo.Interaction,
	answer api.Response,
	flags discordgo.MessageFlags,
) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("%w: %w", ErrSendReply, err)
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, b.config.replyTimeout)
	defer cancel()

	if answer.Content == "" && len(answer.ImageURLs) == 0 {
		return b.sess.InteractionResponseDelete(source, discordgo.WithContext(ctx))
	}

	chunks := splitMessage(answer.Content)
	imageURLs := answer.ImageURLs
	for i := 0; i < len(chunks) || len(imageURLs) > 0; i++ {
		var chunk string
		if i < len(chunks) {
			chunk = chunks[i]
		}
		count := min(len(imageURLs), maxMessageEmbeds)
		var embeds []*discordgo.MessageEmbed
		for _, url := range imageURLs[:count] {
			embeds = append(embeds, &discordgo.MessageEmbed{
				Image: &discordgo.MessageEmbedImage{URL: url},
			})
		}
		imageURLs = imageURLs[count:]

		if i == 0 {
			edit := &discordgo.WebhookEdit{
				Content:         new(chunk),
				AllowedMentions: noMentions(),
			}
			if len(embeds) > 0 {
				edit.Embeds = new(embeds)
			}

			if _, err := b.sess.InteractionResponseEdit(source, edit, discordgo.WithContext(ctx)); err != nil {
				return err
			}
			continue
		}

		if _, err := b.sess.FollowupMessageCreate(source, true, &discordgo.WebhookParams{
			Content:         chunk,
			Embeds:          embeds,
			Flags:           flags,
			AllowedMentions: noMentions(),
		}, discordgo.WithContext(ctx)); err != nil {
			return err
		}
	}

	return nil
}

func noMentions() *discordgo.MessageAllowedMentions {
	return &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}
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
