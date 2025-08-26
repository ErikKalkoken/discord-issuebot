package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Discord command names for interactions
const (
	cmdCreateIssueBug     = "Create bug report"
	cmdCreateIssueFeature = "Create feature request"
)

// Discord custom IDs for interactions
const (
	idCreateIssueRepo  = "createIssueRepo"
	idCreateIssueTitle = "createIssueTitle"
)

type issueType uint

const (
	neutralIssue issueType = iota
	bugReport
	featureRequest
)

func (it issueType) Display() string {
	switch it {
	case neutralIssue:
		return "issue"
	case bugReport:
		return "bug report"
	case featureRequest:
		return "feature request"
	}
	return ""
}

// interactionContext represents a Discord message.
type interactionContext struct {
	authorID         string
	authorName       string
	githubIndex      int
	guildID          string
	channelID        string
	messageID        string
	messageContent   string
	messageTimestamp time.Time
	title            string
	issueType        issueType
}

// Discord commands
var commands = []discordgo.ApplicationCommand{
	{
		Name: cmdCreateIssueBug,
		Type: discordgo.MessageApplicationCommand,
		IntegrationTypes: &[]discordgo.ApplicationIntegrationType{
			discordgo.ApplicationIntegrationUserInstall,
		},
		Contexts: &[]discordgo.InteractionContextType{
			discordgo.InteractionContextBotDM,
			discordgo.InteractionContextPrivateChannel,
			discordgo.InteractionContextGuild,
		},
	},
	{
		Name: cmdCreateIssueFeature,
		Type: discordgo.MessageApplicationCommand,
		IntegrationTypes: &[]discordgo.ApplicationIntegrationType{
			discordgo.ApplicationIntegrationUserInstall,
		},
		Contexts: &[]discordgo.InteractionContextType{
			discordgo.InteractionContextBotDM,
			discordgo.InteractionContextPrivateChannel,
			discordgo.InteractionContextGuild,
		},
	},
}

type Bot struct {
	appID    string
	config   config
	ds       *discordgo.Session
	st       *Storage
	messages sync.Map
	counter  atomic.Int64
}

func NewBot(st *Storage, ds *discordgo.Session, config config) *Bot {
	b := &Bot{
		appID:  config.Discord.AppID,
		config: config,
		ds:     ds,
		st:     st,
	}
	ds.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		slog.Info("Bot is up!")
	})
	ds.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if err := b.handleInteraction(i); err != nil {
			slog.Error("interaction failed", "error", err)
		}
	})
	return b
}

func (b *Bot) InitCommands(isReset bool) error {
	cc, err := b.ds.ApplicationCommands(b.appID, "")
	if err != nil {
		return err
	}
	hasCommands := len(cc) > 0
	if hasCommands && isReset {
		// Delete existing commands
		for _, cmd := range cc {
			err := b.ds.ApplicationCommandDelete(b.appID, "", cmd.ID)
			if err != nil {
				return fmt.Errorf("delete application command %s: %w", cmd.Name, err)
			}
			slog.Info("Deleted application command", "cmd", cmd.Name)
		}
	}
	if !hasCommands || isReset {
		// Add commands
		for _, cmd := range commands {
			_, err := b.ds.ApplicationCommandCreate(b.appID, "", &cmd)
			if err != nil {
				return fmt.Errorf("create application command %s: %w", cmd.Name, err)
			}
			slog.Info("Created application command", "cmd", cmd.Name)
		}
	}
	return nil
}

func (b *Bot) handleInteraction(ic *discordgo.InteractionCreate) error {
	switch ic.Type {
	case discordgo.InteractionApplicationCommand:
		data := ic.ApplicationCommandData()
		createIssue := func(it issueType) error {
			messageID := data.TargetID
			message := data.Resolved.Messages[messageID]
			c := interactionContext{
				authorID:         message.Author.ID,
				authorName:       message.Author.Username,
				channelID:        ic.ChannelID,
				guildID:          ic.GuildID,
				messageContent:   message.Content,
				messageID:        message.ID,
				messageTimestamp: message.Timestamp,
				issueType:        it,
			}
			cid := int(b.counter.Add(1))
			b.messages.Store(cid, c)
			options := make([]discordgo.SelectMenuOption, 0)
			for i, g := range b.config.Github {
				options = append(options, discordgo.SelectMenuOption{
					Label: fmt.Sprintf("%s/%s", g.Owner, g.Repo),
					Value: strconv.Itoa(i),
				})
			}
			err := b.ds.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Create %s [1 / 2]", c.issueType.Display()),
					Flags:   discordgo.MessageFlagsEphemeral,
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									CustomID:    makeCustomID(idCreateIssueRepo, cid),
									Placeholder: "Choose repo",
									Options:     options,
								},
							},
						},
					},
				},
			})
			return err
		}
		name := data.Name
		switch name {
		case cmdCreateIssueBug:
			return createIssue(bugReport)
		case cmdCreateIssueFeature:
			return createIssue(featureRequest)
		}
		return fmt.Errorf("unhandled application command: %s", name)

	case discordgo.InteractionMessageComponent:
		data := ic.MessageComponentData()
		customID := data.CustomID
		if x, found := strings.CutPrefix(customID, idCreateIssueRepo); found {
			cid, err := strconv.Atoi(x)
			if err != nil {
				return err
			}
			x2, ok := b.messages.Load(cid)
			if !ok {
				return fmt.Errorf("failed to load context")
			}
			c := x2.(interactionContext)
			idx, err := strconv.Atoi(data.Values[0])
			if err != nil {
				return err
			}
			c.githubIndex = idx
			b.messages.Store(cid, c)
			err = b.ds.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &discordgo.InteractionResponseData{
					CustomID: makeCustomID(idCreateIssueTitle, cid),
					Title:    fmt.Sprintf("Create %s [2 / 2]", c.issueType.Display()),
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID:  "title",
									Label:     "Title",
									Style:     discordgo.TextInputShort,
									Required:  true,
									MaxLength: 100,
									MinLength: 3,
								},
							},
						},
					},
				},
			})
			return err

		}
		return fmt.Errorf("unhandled message component interaction: %s", customID)

	case discordgo.InteractionModalSubmit:
		data := ic.ModalSubmitData()
		customID := data.CustomID
		if x, found := strings.CutPrefix(customID, idCreateIssueTitle); found {
			cid, err := strconv.Atoi(x)
			if err != nil {
				return err
			}
			x2, ok := b.messages.Load(cid)
			if !ok {
				return fmt.Errorf("failed to load context")
			}
			c := x2.(interactionContext)
			messageURL := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", c.guildID, c.channelID, c.messageID)
			body := fmt.Sprintf(
				"> %s\n\n*Originally posted by **%s** on [Discord](%s)*",
				c.messageContent,
				c.authorName,
				messageURL,
			)
			g := b.config.Github[c.githubIndex]
			var labels []string
			switch c.issueType {
			case bugReport:
				labels = append(labels, "bug")
			case featureRequest:
				labels = append(labels, "enhancement")
			}
			issue, err := createGithubIssue(createGithubIssueParams{
				body:   body,
				labels: labels,
				owner:  g.Owner,
				repo:   g.Repo,
				title:  data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value,
				token:  g.Token,
			})
			if err != nil {
				return err
			}
			err = b.ds.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Issue created on Github\n%s", *issue.HTMLURL),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				return err
			}
			slog.Info(
				"Issue created",
				"repo", fmt.Sprintf("%s/%s", g.Owner, g.Repo),
				"id", *issue.Number,
				"title", *issue.Title,
			)
			return nil
		}
		return fmt.Errorf("unhandled modal submit: %s", customID)
	}

	return fmt.Errorf("unexpected interaction type %d", ic.Type)
}

func makeCustomID(prefix string, id int) string {
	return fmt.Sprintf("%s%d", prefix, id)
}
