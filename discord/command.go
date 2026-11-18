package discord

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/charmingruby/erebor/discord/command"
	"github.com/charmingruby/erebor/o11y/logging"
)

var (
	// ErrInvalidCommand indicates an invalid definition or a duplicate name.
	ErrInvalidCommand = errors.New("invalid Discord command")
	// ErrRegisterCommand indicates a command registration failure.
	ErrRegisterCommand = errors.New("register Discord command")
)

// RegisterCommand adds a local definition. With a command guild, Listen removes
// all global commands and replaces the guild's commands with the local definitions.
// Otherwise, it creates or updates global commands, preserving unrelated commands.
// Duplicate local names are rejected. Register commands before calling Listen.
func (b *Bot) RegisterCommand(definition command.Definition) error {
	if definition.Name == "" || definition.Handler == nil {
		return ErrInvalidCommand
	}

	if _, exists := b.commands[definition.Name]; exists {
		return fmt.Errorf("%w: duplicate name %q", ErrInvalidCommand, definition.Name)
	}

	b.commands[definition.Name] = definition

	return nil
}

func (b *Bot) publishCommands(ctx context.Context) error {
	commands := make([]*discordgo.ApplicationCommand, 0, len(b.commands))

	for _, name := range slices.Sorted(maps.Keys(b.commands)) {
		definition := b.commands[name]
		commands = append(commands, &discordgo.ApplicationCommand{

			Name:        definition.Name,
			Description: definition.Description,
			Type:        discordgo.ChatApplicationCommand,
			Options:     definition.Parameters,
		})
	}

	if b.config.CommandGuildID != "" {
		if err := b.overwriteCommands(ctx, "", []*discordgo.ApplicationCommand{}); err != nil {
			return err
		}

		return b.overwriteCommands(ctx, b.config.CommandGuildID, commands)
	}

	for _, cmd := range commands {
		ctx, cancel := context.WithTimeout(ctx, b.config.HTTPTimeout)

		_, err := b.sess.ApplicationCommandCreate(b.botID(), "", cmd, discordgo.WithContext(ctx))

		cancel()

		if err != nil {
			return fmt.Errorf("%w %q: %w", ErrRegisterCommand, cmd.Name, err)
		}
	}

	return nil
}

func (b *Bot) overwriteCommands(ctx context.Context, guildID string, commands []*discordgo.ApplicationCommand) error {
	ctx, cancel := context.WithTimeout(ctx, b.config.HTTPTimeout)
	defer cancel()

	_, err := b.sess.ApplicationCommandBulkOverwrite(b.botID(), guildID, commands, discordgo.WithContext(ctx))
	if err != nil {
		if guildID == "" {
			return fmt.Errorf("%w: clear global commands: %w", ErrRegisterCommand, err)
		}
		return fmt.Errorf("%w: replace commands in guild %q: %w", ErrRegisterCommand, guildID, err)
	}

	return nil
}

func (b *Bot) handleCommand(ctx context.Context, event *discordgo.InteractionCreate) {
	if event.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data, ok := event.Data.(discordgo.ApplicationCommandInteractionData)
	if !ok || data.CommandType != discordgo.ChatApplicationCommand {
		return
	}

	definition, ok := b.commands[data.Name]
	if !ok || (b.config.CommandGuildID != "" && event.GuildID != b.config.CommandGuildID) {
		return
	}

	log := logging.LoggerFromContext(ctx).With("command", data.Name)
	ctx = logging.WithLogger(ctx, log)

	var flags discordgo.MessageFlags
	if definition.OnlyVisibleToCaller {
		flags = discordgo.MessageFlagsEphemeral
	}

	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: flags,
		},
	}
	if err := b.sendInitialResponse(ctx, event.Interaction, response); err != nil {
		log.ErrorContext(ctx, "respond to Discord interaction", "error", err)
		return
	}

	user := event.User
	if event.Member != nil {
		user = event.Member.User
	}

	handlerCtx, cancel := context.WithTimeout(ctx, min(b.config.HandlerTimeout, 14*time.Minute))
	defer cancel()

	answer, err := definition.Handler(handlerCtx, command.Interaction{
		ID:        event.ID,
		Name:      data.Name,
		GuildID:   event.GuildID,
		ChannelID: event.ChannelID,
		User:      user,
		Member:    event.Member,
		Options:   data.Options,
		Resolved:  data.Resolved,
	})
	cancel()
	if err != nil {
		log.ErrorContext(ctx, "handle Discord command", "error", err)
		answer = b.config.ErrorMessage
	}

	if ctx.Err() != nil {
		return
	}

	if err := b.sendCommandReply(ctx, event.Interaction, answer, flags); err != nil {
		log.ErrorContext(ctx, "respond to Discord interaction", "error", err)
	}
}

func (b *Bot) sendInitialResponse(
	ctx context.Context,
	interaction *discordgo.Interaction,
	response *discordgo.InteractionResponse,
) error {
	// Discord requires the initial response within 3 seconds. Acknowledge before
	// executing the handler.
	ctx, cancel := context.WithTimeout(ctx, min(b.config.HTTPTimeout, 2500*time.Millisecond))
	defer cancel()

	return b.sess.InteractionRespond(interaction, response, discordgo.WithContext(ctx))
}
