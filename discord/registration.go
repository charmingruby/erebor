package discord

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/bwmarrin/discordgo"

	"github.com/charmingruby/erebor/discord/api"
	"github.com/charmingruby/erebor/discord/internal/codec"
)

var (
	// ErrInvalidCommand indicates an invalid definition or a duplicate name.
	ErrInvalidCommand = errors.New("invalid Discord command")
	// ErrRegisterCommand indicates a command registration failure.
	ErrRegisterCommand = errors.New("register Discord command")
)

type registeredCommand struct {
	definition          *discordgo.ApplicationCommand
	handler             api.CommandHandler
	onlyVisibleToCaller bool
}

func (b *Bot) registerCommand(definition api.Command) error {
	registered, err := compileCommand(definition)
	if err != nil {
		return err
	}

	if _, exists := b.commands[definition.Name]; exists {
		return fmt.Errorf("%w: duplicate name %q", ErrInvalidCommand, definition.Name)
	}

	b.commands[definition.Name] = registered

	return nil
}

func (b *Bot) publishCommands(ctx context.Context) error {
	commands := make([]*discordgo.ApplicationCommand, 0, len(b.commands))

	for _, name := range slices.Sorted(maps.Keys(b.commands)) {
		commands = append(commands, b.commands[name].definition)
	}

	if b.config.commandGuildID != "" {
		if err := b.overwriteCommands(ctx, "", []*discordgo.ApplicationCommand{}); err != nil {
			return err
		}

		return b.overwriteCommands(ctx, b.config.commandGuildID, commands)
	}

	for _, cmd := range commands {
		ctx, cancel := context.WithTimeout(ctx, b.config.httpTimeout)

		_, err := b.sess.ApplicationCommandCreate(b.botID(), "", cmd, discordgo.WithContext(ctx))

		cancel()

		if err != nil {
			return fmt.Errorf("%w %q: %w", ErrRegisterCommand, cmd.Name, err)
		}
	}

	return nil
}

func (b *Bot) overwriteCommands(ctx context.Context, guildID string, commands []*discordgo.ApplicationCommand) error {
	ctx, cancel := context.WithTimeout(ctx, b.config.httpTimeout)
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

func compileCommand(def api.Command) (registeredCommand, error) {
	return compileCommandAtDepth(def, 0)
}

func compileCommandAtDepth(def api.Command, depth int) (registeredCommand, error) {
	compiled := registeredCommand{
		definition: &discordgo.ApplicationCommand{
			Name:        def.Name,
			Description: def.Description,
			Type:        discordgo.ChatApplicationCommand,
			Options:     codec.Parameters(def.Parameters),
		},
		handler:             def.Handler,
		onlyVisibleToCaller: def.OnlyVisibleToCaller,
	}

	if len(def.Subcommands) == 0 {
		if def.Handler == nil {
			return registeredCommand{}, fmt.Errorf("%w: command %q needs a handler", ErrInvalidCommand, def.Name)
		}
		return compiled, nil
	}

	if def.Handler != nil || len(def.Parameters) > 0 {
		return registeredCommand{}, fmt.Errorf(
			"%w: command %q cannot combine subcommands with a handler or parameters",
			ErrInvalidCommand,
			def.Name,
		)
	}

	if depth == 2 {
		return registeredCommand{}, fmt.Errorf("%w: command nesting exceeds /command group action", ErrInvalidCommand)
	}

	handlers := make(map[string]api.CommandHandler, len(def.Subcommands))

	for _, sub := range def.Subcommands {
		if _, exists := handlers[sub.Name]; exists {
			return registeredCommand{}, fmt.Errorf(
				"%w: duplicate subcommand %q in %q",
				ErrInvalidCommand,
				sub.Name,
				def.Name,
			)
		}

		child, err := compileCommandAtDepth(sub, depth+1)
		if err != nil {
			return registeredCommand{}, fmt.Errorf("command %q: %w", def.Name, err)
		}

		optionType := discordgo.ApplicationCommandOptionSubCommand
		if len(sub.Subcommands) > 0 {
			optionType = discordgo.ApplicationCommandOptionSubCommandGroup
		}

		handlers[sub.Name] = child.handler
		compiled.definition.Options = append(compiled.definition.Options, &discordgo.ApplicationCommandOption{
			Type:        optionType,
			Name:        sub.Name,
			Description: sub.Description,
			Options:     child.definition.Options,
		})
	}

	compiled.handler = func(ctx context.Context, interaction api.Interaction) (api.Response, error) {
		if len(interaction.Options) != 1 {
			return api.Response{}, fmt.Errorf("%w: expected one subcommand", ErrInvalidCommand)
		}

		sub := interaction.Options[0]
		handle, exists := handlers[sub.Name]

		if !exists {
			return api.Response{}, fmt.Errorf("%w: unknown subcommand %q", ErrInvalidCommand, sub.Name)
		}

		interaction.Options = sub.Options
		interaction.Path = append(interaction.Path, sub.Name)

		return handle(ctx, interaction)
	}

	return compiled, nil
}
