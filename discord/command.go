package discord

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/charmingruby/erebor/discord/api"
	"github.com/charmingruby/erebor/discord/internal/codec"
	"github.com/charmingruby/erebor/o11y/logging"
)

// Command flow: select → acknowledge → execute → deliver.
func (b *Bot) handleCommand(ctx context.Context, event *discordgo.InteractionCreate) {
	if event.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data, ok := event.Data.(discordgo.ApplicationCommandInteractionData)
	if !ok || data.CommandType != discordgo.ChatApplicationCommand {
		return
	}

	definition, ok := b.commands[data.Name]
	if !ok || (b.config.commandGuildID != "" && event.GuildID != b.config.commandGuildID) {
		return
	}

	log := logging.LoggerFromContext(ctx).With("command", data.Name)
	ctx = logging.WithLogger(ctx, log)

	var flags discordgo.MessageFlags
	if definition.onlyVisibleToCaller {
		flags = discordgo.MessageFlagsEphemeral
	}

	if err := b.acknowledgeCommand(ctx, event.Interaction, flags); err != nil {
		log.ErrorContext(ctx, "acknowledge Discord command", "error", err)
		return
	}

	answer, err := b.executeCommand(ctx, definition.handler, codec.Interaction(event, data))
	if err != nil {
		log.ErrorContext(ctx, "handle Discord command", "error", err)
		answer = api.Response{Content: b.config.errorMessage}
	}

	if ctx.Err() != nil {
		return
	}

	if err := b.sendCommandReply(ctx, event.Interaction, answer, flags); err != nil {
		log.ErrorContext(ctx, "reply to Discord command", "error", err)
	}
}

func (b *Bot) executeCommand(
	ctx context.Context,
	handler api.CommandHandler,
	input api.Interaction,
) (api.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, min(b.config.handlerTimeout, 14*time.Minute))
	defer cancel()

	return handler(ctx, input)
}

func (b *Bot) acknowledgeCommand(
	ctx context.Context,
	interaction *discordgo.Interaction,
	flags discordgo.MessageFlags,
) error {
	// Discord requires acknowledgment within 3 seconds, before running user code.
	ctx, cancel := context.WithTimeout(ctx, min(b.config.httpTimeout, 2500*time.Millisecond))
	defer cancel()

	return b.sess.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: flags},
	}, discordgo.WithContext(ctx))
}
