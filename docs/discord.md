# Discord

A reusable bot built on DiscordGo, supporting slash commands and optional
mentions. Application logic stays in callbacks; the package has no dependency on
AI models or application prompts.

## Slash commands

Mention handling is opt-in. Register commands before starting the listener:

```go
// Imports: github.com/charmingruby/erebor/discord
//          github.com/charmingruby/erebor/discord/command

bot, err := discord.NewBot(
    os.Getenv("DISCORD_BOT_TOKEN"),
    discord.WithCommandGuild(os.Getenv("DISCORD_GUILD_ID")), // Empty = global.
    discord.WithErrorMessage("Não consegui executar o comando. Tente novamente."),
)
if err != nil {
    return err
}

err = bot.RegisterCommand(command.Definition{
    Name: "echo",
    Description: "Retorna o texto informado",
    OnlyVisibleToCaller: true,
    Parameters: []*command.Parameter{
        command.String(command.Parameter{
            Name: "text",
            Description: "Texto para retornar",
            Required: true,
        }),
    },
    Handler: func(ctx context.Context, interaction command.Interaction) (string, error) {
        for _, option := range interaction.Options {
            if option.Name == "text" {
                return option.StringValue(), nil
            }
        }
        return "", nil
    },
})
if err != nil {
    return err
}

ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
return bot.Listen(ctx)
```

Build parameters with `command.String`, `command.Integer`, `command.Number`,
`command.Boolean`, `command.User`, `command.Role`, `command.Channel`, `command.Mentionable`,
or `command.Attachment`. Each accepts a `command.Parameter` with explicit fields
and sets its Discord type. Parameters are optional unless `Required: true` is set.
String, integer, and number parameters support `Choices`.
Use `command.Subcommand` and `SubcommandGroup` for nested actions:

```go
parameters := []*command.Parameter{
    command.Subcommand("set", "Define uma preferência",
        command.String(command.Parameter{
            Name: "language",
            Description: "Idioma",
            Required: true,
            Choices: []*command.Choice{
                {Name: "Português", Value: "pt"},
                {Name: "English", Value: "en"},
            },
        }),
    ),
}
```

No DiscordGo import is needed for these definitions. `Parameter` and `Choice` are
aliases, so advanced DiscordGo fields remain available. The callback gets
`Options`, `Resolved` objects, `User`, `Member`, `ChannelID`, and `GuildID`.
`Member` may be nil in DMs. Autocomplete and context-menu commands are not handled.

- Install with the `bot` and `applications.commands` OAuth scopes. Command-only
  bots do not need Message Content Intent.
- With a nonempty `WithCommandGuild`, `Listen` deletes **all global commands**
  for the application, then bulk overwrites the configured guild's commands with
  the current definitions. Removed or renamed commands disappear; an empty list
  clears that guild's commands. Commands in other guilds are not removed.
  **Do not share the application with production if it relies on global commands.**
- Without a command guild, `Listen` creates or updates global commands individually,
  preserving unrelated or removed commands. Global commands may take time to appear.
- Registration failures close the connection without rolling back earlier updates,
  including global command cleanup.
- Invocations are deferred before the callback, with a 2.5-second acknowledgment
  timeout. Failed acknowledgments skip execution. `OnlyVisibleToCaller` hides
  every response chunk from users other than the caller.
- Empty results remove the deferred response. Failures use `ErrorMessage`, or
  remove the response when no fallback is configured.
- Register commands before listening; duplicate names are rejected.
  Do not mutate registered parameter definitions.

## Mentions

To enable mentions and replies to bot messages, supply `WithMentionReplyHandler`:

```go
// Import github.com/charmingruby/erebor/discord/mention.
discord.WithMentionReplyHandler(func(ctx context.Context, message mention.Message) (string, error) {
    return "Você disse: " + message.Content, nil
})
```

Enable **Message Content Intent** and grant permissions to view channels, send
messages, and read message history. Mentions and replies from guild users are
handled; bots and DMs are ignored. Empty mentions reach the callback. Typing is
refreshed every 8 seconds, and empty answers send nothing.

`ReferencedContent` holds the previous bot reply. Treat it as untrusted data,
not instructions, when building AI prompts.

## Package boundaries

- `discord` owns the connection, registration, shutdown, and reply delivery.
- `discord/command` owns slash-command definitions and parameter helpers.
- `discord/mention` owns mention/reply callbacks and message data.

The root package imports the capability packages, never the reverse. Callbacks
receive only their input and context; passing the whole bot is unnecessary.
Applications can organize their own capabilities in packages that return
`command.Definition` or `mention.Handler`.

## Shared behavior

- Both features work together or independently. Supply a mention handler or at
  least one command.
- Defaults: 15-second HTTP timeout, 3-minute handler timeout, and 1-minute reply timeout.
- Override these with `WithHTTPTimeout`,
  `WithHandlerTimeout`, and `WithReplyTimeout`. Explicit values must be positive;
  omitted options use defaults. Later options override earlier ones.
- Message whitespace is preserved; only mentions of this bot are removed.
  Command payload rules are validated by Discord during registration.
- Handlers must honor cancellation. Command execution is capped at 14 minutes
  for Discord's token lifetime. Shutdown cancels handlers and waits for them.
- Handlers may run concurrently without a package-level limit. Applications
  should limit expensive operations when needed; timeouts do not limit concurrency.
- Replies use chunks of at most 2,000 UTF-16 units, preserving Unicode and
  disabling generated mentions. Only the first mention-reply chunk quotes its source.
- Handler failures are logged and optionally send `ErrorMessage`.
- Normal shutdown returns `nil`; an already-canceled context returns its error.
  Configure and register commands sequentially before calling `Listen`, and call
  it only once per bot. Concurrent registration, configuration during execution,
  and repeated or concurrent listeners are unsupported.
