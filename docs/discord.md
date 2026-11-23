# Discord

Bots with slash commands, subcommands, mentions, and message replies.
Applications use only Erebor types; `discordgo` is an internal implementation detail.

## Create and run

`New(token, options...)` applies options and validates once, without connecting.
`Listen` connects, publishes commands, and processes events until cancellation.
All public contracts live together in `discord/api`.

```go
import (
    "context"
    "os"
    "os/signal"

    "github.com/charmingruby/erebor/discord"
    "github.com/charmingruby/erebor/discord/api"
)

func run() error {
    bot, err := discord.New(os.Getenv("DISCORD_BOT_TOKEN"),
        discord.WithErrorMessage("Unable to execute the command."),
        discord.WithCommands(api.Command{
            Name: "echo",
            Description: "Returns the supplied text",
            OnlyVisibleToCaller: true,
            Parameters: []api.Parameter{
                {
                    Type: api.ParameterString,
                    Name: "text",
                    Description: "Text to return",
                    Required: true,
                },
            },
            Handler: func(ctx context.Context, i api.Interaction) (api.Response, error) {
                return api.Response{Content: i.String("text")}, nil
            },
        }),
    )
    if err != nil {
        return err
    }
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
    defer stop()
    return bot.Listen(ctx)
}
```

Install the bot with the `bot` and `applications.commands` OAuth scopes.
Command-only bots do not need Message Content Intent.

## Parameters and data

The `Type` field uses constants from `api`: `ParameterString`, `ParameterInteger`,
`ParameterNumber`, `ParameterBoolean`, `ParameterUser`, `ParameterRole`,
`ParameterChannel`, `ParameterMentionable`, and `ParameterAttachment`.
Parameters are optional unless `Required: true` is set.

- `Choices: []api.Choice` restricts string, integer, or number values.
- `MinValue`, `MaxValue`, `MinLength`, and `MaxLength` define bounds.
- `ChannelTypes: []api.ChannelType` restricts channel types, such as
  `api.GuildText` and `api.GuildVoice`.
- Localizations use `map[string]string`, with locale codes such as `"pt-BR"`.
- Discord validates naming rules and limits when commands are published.
  Autocomplete and context-menu commands are not part of this API.

In callbacks, read arguments with `i.String(name)`, `i.Integer(name)`,
`i.Number(name)`, and `i.Boolean(name)`. Missing arguments return zero values;
`i.GetOption(name)` returns `nil`, letting you distinguish absence from zero.
User, channel, role, mentionable, and attachment arguments return IDs through
`i.String(name)`. Their objects are available in the maps on `i.Resolved`:

```go
id := i.String("file")
if i.Resolved != nil {
    if file := i.Resolved.Attachments[id]; file != nil {
        return api.Response{Content: file.URL}, nil
    }
}
```

`i.User` is the invoking user; `i.Member` is their guild membership (nil in DMs).
Entities in `discord/api` are library-owned data types, not sessions or
objects with network methods.

## Subcommands

Use `Subcommands: []api.Command` instead of `Handler` and `Parameters`.
Each action has its own parameters and handler. For `/tools echo`:

```go
api.Command{
    Name: "tools",
    Description: "Utilities",
    Subcommands: []api.Command{
        {
            Name: "echo",
            Description: "Returns text",
            Parameters: []api.Parameter{
                {
                    Type: api.ParameterString,
                    Name: "text", Description: "Text", Required: true,
                },
            },
            Handler: func(ctx context.Context, i api.Interaction) (api.Response, error) {
                return api.Response{Content: i.String("text")}, nil
            },
        },
    },
}
```

You can nest a group: `/command group action`. The handler receives only
the selected action's arguments. `i.Name` remains the root command name, and
`i.Path` identifies the full path. Set `OnlyVisibleToCaller` on the root command.

## Mentions and replies

Use `WithMentionHandler` to handle mentions and replies to the bot's messages:

```go
discord.WithMentionHandler(func(ctx context.Context, m api.Mention) (string, error) {
    return "You said: " + m.Content, nil
})
```

Enable **Message Content Intent** and permissions to view channels, send
messages, and read message history. Bots and DMs are ignored. Content excludes
mentions of this bot without changing other whitespace. For replies,
`ReferencedContent` contains the previous bot message. Treat it as untrusted data.
The typing indicator is refreshed while the handler runs.

## Responses, errors, and shutdown

- `api.Response{Content: "...", ImageURL: "https://..."}` sends text and/or
  a public image. Images appear in the first chunk.
- Commands are acknowledged before the handler runs. An empty response removes
  the deferred response; for mentions, an empty response sends nothing.
- Text is split into chunks of at most 2,000 UTF-16 units without splitting characters.
  Generated mentions are disabled. Private visibility applies to every chunk.
- Handler errors are logged and use `WithErrorMessage` as a fallback.
  Without a fallback, the response is empty.
- Defaults are 15 seconds per request, 3 minutes per handler, and 1 minute
  for reply delivery. Use `WithHTTPTimeout`, `WithHandlerTimeout`, and
  `WithReplyTimeout` with positive values to override them.
  Command execution is capped at 14 minutes because of Discord's token lifetime.
- Handlers may run concurrently and must honor context cancellation.
  On shutdown, the library cancels handlers and waits for them to finish.
- Call `Listen` once per bot. Normal cancellation returns `nil`;
  an already-canceled context returns its error.

## Publishing commands

Without `WithCommandGuild`, global commands are created or updated individually;
old or unrelated commands are preserved.

**With a nonempty `WithCommandGuild`, all global commands for the application are
deleted, and the selected guild's commands are replaced.** This mode is intended
for development with an application separate from production. Other guilds
are not changed. Failures do not roll back earlier updates.

## Maintenance and migration

- `bot.go`: construction, connection, and shutdown.
- `registration.go`: compilation, subcommand routing, and publishing.
- `command.go`: selection → acknowledgment → execution → reply.
- `message.go`: mention/reply selection → execution with typing → reply.
- `reply.go`: delivery, text splitting, and mention safety.
- `internal/codec`: conversion between DiscordGo and public contracts.
- `api`: application-facing commands, parameters, responses, mentions, and entities.

Supply commands with `WithCommands`, which accepts multiple commands and appends
them across repeated calls. Other options use the last supplied value.
Options do not validate individually; `New` validates the final result.

Local checks are limited to the token, timeouts, handler presence,
duplicate names, ambiguous commands, and subcommand depth. Discord-specific
parameter and naming rules are checked by Discord only when publishing.

Do not mutate definitions after creating the bot. There is no runtime registration,
generic pointer copying, or repeated validation in `Listen`.

To migrate from `New(Config)`, pass the token as the first argument and use
`WithCommands`, `WithMentionHandler`, `WithErrorMessage`, and
`WithCommandGuild`. Replace imports of `command`, `mention`, and `model` with
`discord/api`, and declare each parameter's type in its `Type` field.
