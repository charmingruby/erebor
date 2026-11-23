package api

import "context"

// Command describes a slash command, subcommand, or subcommand group.
// Do not mutate definitions after supplying them to a bot.
type Command struct {
	Name        string
	Description string
	Handler     CommandHandler
	Parameters  []Parameter
	// Subcommands defines actions with their own handlers and parameters.
	// A command with subcommands cannot also have a Handler or Parameters.
	Subcommands []Command
	// OnlyVisibleToCaller hides every response chunk from other users.
	// Set it on the root command, not on individual subcommands.
	OnlyVisibleToCaller bool
}

// CommandHandler processes a command. It may run concurrently and must honor context
// cancellation. An empty Response removes the deferred response.
type CommandHandler func(context.Context, Interaction) (Response, error)

// Interaction contains command arguments and the invoking user and channel.
// Options contains only the parameters of the command or subcommand being handled.
// Name remains the root command name. Resolved holds referenced objects.
// Member may be nil in direct messages.
type Interaction struct {
	ID        string
	Name      string
	ChannelID string
	GuildID   string
	User      *User
	Member    *Member
	Resolved  *Resolved
	Options   []*Argument
	// Path identifies the selected action, e.g. ["tools", "admin", "ban"].
	Path []string
}
