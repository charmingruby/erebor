package discord

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/charmingruby/erebor/o11y/logging"
)

var (
	// ErrInvalidCommand indicates an invalid definition or a duplicate name.
	ErrInvalidCommand = errors.New("invalid Discord command")
	// ErrRegisterCommand indicates a command registration failure.
	ErrRegisterCommand = errors.New("register Discord command")
)

// Command defines a slash command. Register it before Listen. Options support
// DiscordGo's parameter types, choices, subcommands, and subcommand groups.
// Do not mutate Options or their contents after registration.
type Command struct {
	Name        string
	Description string
	Handler     CommandHandler
	Options     []*discordgo.ApplicationCommandOption
	// Ephemeral makes all responses visible only to the invoking user.
	// Visibility must be chosen before the interaction is acknowledged.
	Ephemeral bool
}

// CommandHandler processes a command. It must honor context cancellation and
// may run concurrently. Empty output removes the deferred response.
type CommandHandler func(context.Context, Interaction) (string, error)

// Interaction contains command arguments and the invoking user and channel.
// Options retain their hierarchy for subcommands; Resolved contains objects
// referenced by user, role, channel, and attachment parameters.
type Interaction struct {
	ID        string
	Name      string
	ChannelID string
	GuildID   string
	User      *discordgo.User
	Member    *discordgo.Member
	Resolved  *discordgo.ApplicationCommandInteractionDataResolved
	Options   []*discordgo.ApplicationCommandInteractionDataOption
}

// RegisterCommand adds a local definition. Listen creates or updates commands
// individually in Discord, preserving unrelated commands. Duplicate local names
// are rejected, and registration while listening returns ErrAlreadyRunning.
func (b *Bot) RegisterCommand(command Command) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running {
		return ErrAlreadyRunning
	}

	if command.Name == "" || command.Handler == nil {
		return ErrInvalidCommand
	}

	if _, exists := b.commands[command.Name]; exists {
		return fmt.Errorf("%w: duplicate name %q", ErrInvalidCommand, command.Name)
	}

	b.commands[command.Name] = command

	return nil
}

func (b *Bot) publishCommands(ctx context.Context) error {
	for _, name := range slices.Sorted(maps.Keys(b.commands)) {
		command := b.commands[name]

		ctx, cancel := context.WithTimeout(ctx, b.config.RequestTimeout)

		_, err := b.sess.ApplicationCommandCreate(b.botID(), b.config.CommandGuildID,
			&discordgo.ApplicationCommand{
				Name:        command.Name,
				Description: command.Description,
				Type:        discordgo.ChatApplicationCommand,
				Options:     command.Options,
			}, discordgo.WithContext(ctx))

		cancel()

		if err != nil {
			return fmt.Errorf("%w %q: %w", ErrRegisterCommand, name, err)
		}
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

	command, ok := b.commands[data.Name]
	if !ok || (b.config.CommandGuildID != "" && event.GuildID != b.config.CommandGuildID) {
		return
	}

	log := logging.LoggerFromContext(ctx).With("command", data.Name)
	ctx = logging.WithLogger(ctx, log)

	if !b.tryAcquireSlot() {
		if err := b.sendInteractionResponse(ctx, event.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "The bot is busy. Please try again shortly.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
			log.ErrorContext(ctx, "respond to Discord interaction", "error", err)
		}

		return
	}

	defer func() {
		<-b.slots
	}()

	var flags discordgo.MessageFlags
	if command.Ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}

	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: flags,
		},
	}
	if err := b.sendInteractionResponse(ctx, event.Interaction, response); err != nil {
		log.ErrorContext(ctx, "respond to Discord interaction", "error", err)
		return
	}

	user := event.User
	if event.Member != nil {
		user = event.Member.User
	}

	handlerCtx, cancel := context.WithTimeout(ctx, min(b.config.HandlerTimeout, 14*time.Minute))
	defer cancel()

	answer, err := command.Handler(handlerCtx, Interaction{
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

	if err := b.respondToCommand(ctx, event.Interaction, answer, flags); err != nil {
		log.ErrorContext(ctx, "respond to Discord interaction", "error", err)
	}
}

func (b *Bot) sendInteractionResponse(
	ctx context.Context,
	interaction *discordgo.Interaction,
	response *discordgo.InteractionResponse,
) error {
	// Discord requires the initial response within 3 seconds. Never wait for a
	// handler or a concurrency slot before acknowledging.
	ctx, cancel := context.WithTimeout(ctx, min(b.config.RequestTimeout, 2500*time.Millisecond))
	defer cancel()

	return b.sess.InteractionRespond(interaction, response, discordgo.WithContext(ctx))
}
