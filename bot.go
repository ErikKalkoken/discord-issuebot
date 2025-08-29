package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	ErrInvalidURL       = errors.New("invalid repo URL")
	ErrInvalidArguments = errors.New("invalid arguments")
)

// Discord command names for interactions
const (
	cmdIssueCreateBug     = "Create bug report"
	cmdIssueCreateFeature = "Create feature request"
	cmdManage             = "issuebot"
)

// Discord custom IDs for interactions
const (
	idIssueCreateIssue1 = "issueCreateIssue1-"
	idIssueCreateIssue2 = "issueCreateIssue2-"
	idRepoAdd1          = "repoAdd1"
	idRepoAdd2          = "repoAdd2-"
	idRepoDelete        = "repoDelete-"
	idRepoTest          = "repoTest-"
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

// createIssueData represents the data of an interaction session.
type createIssueData struct {
	authorID         string
	authorName       string
	channelID        string
	guildID          string
	issueType        issueType
	messageContent   string
	messageID        string
	messageTimestamp time.Time
	repoID           int
	title            string
}

// Discord commands
var commands = []discordgo.ApplicationCommand{
	{
		Name:        cmdManage,
		Description: "Manage repositories",
		Type:        discordgo.ChatApplicationCommand,
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
		Name: cmdIssueCreateBug,
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
		Name: cmdIssueCreateFeature,
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
	api      *repoAPI
	appID    string
	counter  atomic.Int64
	ds       *discordgo.Session
	sessions sync.Map
	st       *Storage
}

func NewBot(st *Storage, ds *discordgo.Session, appID string) *Bot {
	b := &Bot{
		api:   newRepoAPI(),
		appID: appID,
		ds:    ds,
		st:    st,
	}
	ds.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		slog.Info("Bot is up", "appID", appID)
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
	respondWithMessage := func(content string) error {
		err := b.ds.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return err
	}

	var userID string
	if ic.Member != nil && ic.Member.User != nil {
		userID = ic.Member.User.ID
	} else if ic.User != nil {
		userID = ic.User.ID
	} else {
		return fmt.Errorf("no user found for interaction")
	}

	switch ic.Type {
	case discordgo.InteractionApplicationCommand:
		data := ic.ApplicationCommandData()
		createIssue := func(it issueType) error {
			messageID := data.TargetID
			message := data.Resolved.Messages[messageID]
			s := createIssueData{
				authorID:         message.Author.ID,
				authorName:       message.Author.Username,
				channelID:        ic.ChannelID,
				guildID:          ic.GuildID,
				issueType:        it,
				messageContent:   message.Content,
				messageID:        message.ID,
				messageTimestamp: message.Timestamp,
			}
			sessionID := b.newSessionID()
			b.sessions.Store(sessionID, s)
			repos, err := b.st.ListReposForUser(userID)
			if err != nil {
				return err
			}
			if len(repos) == 0 {
				return respondWithMessage(":exclamation: Please add a repo")
			}
			options := make([]discordgo.SelectMenuOption, 0)
			for _, r := range repos {
				options = append(options, discordgo.SelectMenuOption{
					Label: r.Name(),
					Value: strconv.Itoa(r.ID),
				})
			}
			err = b.ds.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Create %s [1 / 2]", s.issueType.Display()),
					Flags:   discordgo.MessageFlagsEphemeral,
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									CustomID:    idIssueCreateIssue1 + sessionID,
									Options:     options,
									Placeholder: "Choose repo",
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

		case cmdIssueCreateBug:
			return createIssue(bugReport)

		case cmdIssueCreateFeature:
			return createIssue(featureRequest)

		case cmdManage:
			err := b.ds.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				return err
			}
			repos, err := b.st.ListReposForUser(userID)
			if err != nil {
				return err
			}
			components := []discordgo.MessageComponent{discordgo.TextDisplay{
				Content: fmt.Sprintf("%d repos", len(repos)),
			}}
			for _, r := range repos {
				container := discordgo.Container{
					Components: []discordgo.MessageComponent{
						discordgo.TextDisplay{
							Content: fmt.Sprintf("[%s](%s)", r.Name(), r.URL()),
						},
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									CustomID: fmt.Sprintf("%s%d", idRepoDelete, r.ID),
									Label:    "Delete",
									Style:    discordgo.DangerButton,
								},
								discordgo.Button{
									CustomID: fmt.Sprintf("%s%d", idRepoTest, r.ID),
									Label:    "Test",
								},
							},
						},
					},
				}
				components = append(components, container)
			}

			components = append(components, discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: idRepoAdd1,
						Label:    "Add repository",
					},
				},
			})

			for chunk := range slices.Chunk(components, 30) {
				_, err = b.ds.FollowupMessageCreate(ic.Interaction, false, &discordgo.WebhookParams{
					Flags:      discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2,
					Components: chunk,
				})
				if err != nil {
					return err
				}
			}
			return nil
		}
		return fmt.Errorf("unhandled application command: %s", name)

	case discordgo.InteractionMessageComponent:
		data := ic.MessageComponentData()
		customID := data.CustomID

		if customID == idRepoAdd1 {
			err := b.ds.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &discordgo.InteractionResponseData{
					CustomID: idRepoAdd2 + userID,
					Title:    "Add repo",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID:    "url",
									Label:       "Repository URL",
									Placeholder: "https://github.com/{OWNER}/{REPO}",
									Required:    true,
									Style:       discordgo.TextInputShort,
								},
							},
						},
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID:    "token",
									Label:       "Token",
									Placeholder: "Github issues read & write access",
									Required:    true,
									Style:       discordgo.TextInputShort,
								},
							},
						},
					},
				},
			})
			return err

		} else if sessionID, found := strings.CutPrefix(customID, idIssueCreateIssue1); found {
			x2, ok := b.sessions.Load(sessionID)
			if !ok {
				return fmt.Errorf("failed to load session")
			}
			s := x2.(createIssueData)
			idx, err := strconv.Atoi(data.Values[0])
			if err != nil {
				return err
			}
			s.repoID = idx
			b.sessions.Store(sessionID, s)
			err = b.ds.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &discordgo.InteractionResponseData{
					CustomID: idIssueCreateIssue2 + sessionID,
					Title:    fmt.Sprintf("Create %s [2 / 2]", s.issueType.Display()),
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID: "title",
									Label:    "Title",
									Style:    discordgo.TextInputShort,
									Required: true,
								},
							},
						},
					},
				},
			})
			return err

		} else if x, found := strings.CutPrefix(customID, idRepoDelete); found {
			repoID, err := strconv.Atoi(x)
			if err != nil {
				return err
			}
			err = b.st.DeleteRepo(repoID)
			if err != nil {
				return err
			}
			err = b.ds.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2,
					Components: []discordgo.MessageComponent{
						discordgo.TextDisplay{
							Content: ":white_check_mark: Repo deleted",
						},
					},
				},
			})
			return err

		} else if x, found := strings.CutPrefix(customID, idRepoTest); found {
			repoID, err := strconv.Atoi(x)
			if err != nil {
				return err
			}
			r, err := b.st.GetRepo(repoID)
			if err != nil {
				return err
			}
			var s string
			status, err := b.api.checkToken(r)
			if err != nil {
				slog.Warn("Failed to get info for github repo", "error", err)
				var m string
				switch status {
				case http.StatusUnauthorized:
					m = "Invalid token"
				case http.StatusNotFound:
					m = "Repository not found"
				default:
					m = "Internal error"
				}
				s = fmt.Sprintf(":x: %s: Test failed: %s", r.Name(), m)
			} else {
				s = fmt.Sprintf(":white_check_mark: %s: Test succeeded", r.Name())
			}
			err = b.ds.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2,
					Components: []discordgo.MessageComponent{
						discordgo.TextDisplay{
							Content: s,
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

		if sessionID, found := strings.CutPrefix(customID, idIssueCreateIssue2); found {
			x, ok := b.sessions.Load(sessionID)
			if !ok {
				return fmt.Errorf("failed to load session")
			}
			s := x.(createIssueData)
			messageURL := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", s.guildID, s.channelID, s.messageID)
			body := fmt.Sprintf(
				"> %s\n\n*Originally posted by **%s** on [Discord](%s)*",
				s.messageContent,
				s.authorName,
				messageURL,
			)
			r, err := b.st.GetRepo(s.repoID)
			if err != nil {
				return err
			}
			var labels []string
			switch s.issueType {
			case bugReport:
				labels = append(labels, "bug")
			case featureRequest:
				labels = append(labels, "enhancement")
			}

			title := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
			htmlURL, err := b.api.createIssue(r, createIssueParams{
				body:   body,
				labels: labels,
				title:  title,
			})
			if err != nil {
				return err
			}
			slog.Info("Issue created", "repo", r.Name(), "title", title, "url", htmlURL)
			err = b.ds.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf(":white_check_mark: Issue created on Github\n%s", htmlURL),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				return err
			}
			b.sessions.Delete(sessionID)
			return nil

		} else if userID, found := strings.CutPrefix(customID, idRepoAdd2); found {
			err := b.ds.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				return err
			}
			rawURL := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
			owner, repo, vendor, err := parseRepoURL(rawURL)
			if err != nil {
				slog.Warn("Failed to parse URL", "url", rawURL, "error", err)
				_, err2 := b.ds.FollowupMessageCreate(ic.Interaction, false, &discordgo.WebhookParams{
					Content: ":x: Failed to add repo: " + err.Error(),
				})
				if err2 != nil {
					return err2
				}
				return nil
			}
			token := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
			status, err := b.api.checkToken(&Repo{
				Owner:  owner,
				Repo:   repo,
				Token:  token,
				UserID: userID,
				Vendor: vendor,
			})
			if err != nil {
				slog.Warn("Failed to get info for github repo", "error", err)
				var m string
				switch status {
				case http.StatusUnauthorized:
					m = "Invalid token"
				case http.StatusNotFound:
					m = "Repository not found"
				default:
					m = "Internal error"
				}
				_, err2 := b.ds.FollowupMessageCreate(ic.Interaction, false, &discordgo.WebhookParams{
					Content: ":x: Failed to add repo: " + m,
				})
				if err2 != nil {
					return err2
				}
				return nil
			}
			r, created, err := b.st.UpdateOrCreateRepo(UpdateOrCreateRepoParams{
				UserID: userID,
				Owner:  owner,
				Repo:   repo,
				Token:  token,
				Vendor: vendor,
			})
			if err != nil {
				return err
			}
			var action string
			if created {
				action = "added"
			} else {
				action = "updated"
			}
			_, err = b.ds.FollowupMessageCreate(ic.Interaction, false, &discordgo.WebhookParams{
				Content: fmt.Sprintf(":white_check_mark: Repo %s: %s", action, r.Name()),
			})
			if err != nil {
				return err
			}
			return nil
		}
		return fmt.Errorf("unhandled modal submit: %s", customID)
	}

	return fmt.Errorf("unexpected interaction type %d", ic.Type)
}

func (b *Bot) newSessionID() string {
	return strconv.Itoa(int(b.counter.Add(1)))
}

func parseRepoURL(s string) (string, string, Vendor, error) {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return "", "", undefined, err
	}
	var v Vendor
	switch u.Host {
	case "github.com":
		v = gitHub
	case "gitlab.com":
		v = gitLab
	default:
		return "", "", undefined, fmt.Errorf("host must be github.com or gitlab.com: %w", ErrInvalidURL)
	}
	x := strings.Split(u.Path, "/")
	if len(x) != 3 {
		return "", "", undefined, fmt.Errorf("path must have exactly two parts: %w", ErrInvalidURL)
	}
	return x[1], x[2], v, nil
}
