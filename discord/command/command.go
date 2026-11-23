package command

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

// Definition describes a slash command, subcommand, or subcommand group.
// Do not mutate its parameters or subcommands after registration.
type Definition struct {
	Name        string
	Description string
	Handler     Handler
	Parameters  []*Parameter
	// Subcommands defines actions with their own handlers and parameters.
	// A command with subcommands cannot also have a Handler or Parameters.
	Subcommands []Definition
	// OnlyVisibleToCaller hides every response chunk from other users.
	// Set it on the root command, not on individual subcommands.
	OnlyVisibleToCaller bool
}

// Handler processes a command. It may run concurrently and must honor context
// cancellation. An empty Response removes the deferred response.
type Handler func(context.Context, Interaction) (Response, error)

// Response contains the reply text and an optional image displayed below it.
// ImageURL must be a publicly accessible HTTP(S) image URL.
type Response struct {
	Content  string
	ImageURL string
}

// Interaction contains command arguments and the invoking user and channel.
// Options contains only the parameters of the command or subcommand being handled.
// Name remains the root command name. Resolved holds referenced objects.
// Member may be nil in direct messages.
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
