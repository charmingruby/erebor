package discord

import (
	"context"
	"fmt"
	"net/http"
	"unicode/utf16"

	"github.com/bwmarrin/discordgo"

	"github.com/charmingruby/erebor/discord/api"
)

const (
	maxMessageUnits = 2000
	maxGalleryItems = 10
)

// sendMessageReply delivers the text returned by handleMentionOrReply.
// Only the first chunk references the user's message; all chunks disable mentions
// and share a single reply timeout. Mention replies do not use Components V2.
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

// sendCommandReply delivers the response returned by handleCommand.
// It splits text and images into message-sized batches, builds their components,
// edits the initial deferred response, then sends any remaining batches as
// follow-ups with the original visibility. All requests share one reply timeout;
// an empty answer deletes the deferred response instead.
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

		count := min(len(imageURLs), maxGalleryItems)
		components := buildCommandReplyComponents(chunk, imageURLs[:count])
		imageURLs = imageURLs[count:]

		if i == 0 {
			if err := b.editInitialCommandReply(ctx, source, components); err != nil {
				return err
			}

			continue
		}

		if _, err := b.sess.FollowupMessageCreate(source, true, &discordgo.WebhookParams{
			Components:      components,
			Flags:           flags | discordgo.MessageFlagsIsComponentsV2,
			AllowedMentions: noMentions(),
		}, discordgo.WithContext(ctx)); err != nil {
			return err
		}
	}

	return nil
}

// buildCommandReplyComponents is the payload builder used by sendCommandReply
// for both the initial response and follow-ups. It places nonempty text before
// the optional gallery, without sending requests or modifying the inputs.
// sendCommandReply supplies one text chunk and at most maxGalleryItems URLs.
func buildCommandReplyComponents(text string, imageURLs []string) []discordgo.MessageComponent {
	var components []discordgo.MessageComponent

	if text != "" {
		components = append(components, discordgo.TextDisplay{Content: text})
	}

	var items []discordgo.MediaGalleryItem

	for _, url := range imageURLs {
		items = append(items, discordgo.MediaGalleryItem{
			Media: discordgo.UnfurledMediaItem{URL: url},
		})
	}

	if len(items) > 0 {
		components = append(components, discordgo.MediaGallery{Items: items})
	}

	return components
}

// editInitialCommandReply is used by sendCommandReply for its first batch only.
// It replaces the deferred response with Components V2; the private/public
// visibility set during acknowledgment is preserved. The caller owns the timeout
// and error wrapping. Subsequent batches use FollowupMessageCreate instead.
//
// DiscordGo v0.29.0's WebhookEdit does not expose Flags, so use its REST client
// to set the required flag. Replace this with InteractionResponseEdit once
// WebhookEdit supports Flags.
func (b *Bot) editInitialCommandReply(
	ctx context.Context,
	source *discordgo.Interaction,
	components []discordgo.MessageComponent,
) error {
	edit := struct {
		AllowedMentions *discordgo.MessageAllowedMentions `json:"allowed_mentions"`
		Components      []discordgo.MessageComponent      `json:"components"`
		Flags           discordgo.MessageFlags            `json:"flags"`
	}{
		Components:      components,
		Flags:           discordgo.MessageFlagsIsComponentsV2,
		AllowedMentions: noMentions(),
	}
	endpoint := discordgo.EndpointWebhookMessage(source.AppID, source.Token, "@original")
	_, err := b.sess.RequestWithBucketID(http.MethodPatch, endpoint, edit, endpoint, discordgo.WithContext(ctx))

	return err
}

// noMentions disables automatic mention parsing for both delivery paths.
func noMentions() *discordgo.MessageAllowedMentions {
	return &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}
}

// splitMessage is shared by mention and command replies. It splits text at
// Discord's UTF-16 message limit without breaking a Unicode character.
// Empty input produces no chunks.
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
