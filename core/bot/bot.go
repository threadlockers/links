package bot

import (
	"log"
	"net/url"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/threadlockers/links/core/config"
	"github.com/threadlockers/links/core/helpers"
	"github.com/threadlockers/links/core/utils"
)

type Bot struct {
	session *discordgo.Session
	cfg     config.EnvCfg
}

func New(cfg config.EnvCfg) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.DiscordBotToken)
	if err != nil {
		return nil, err
	}

	session.Identify.Intents = discordgo.IntentGuildMessages |
		discordgo.IntentGuildMessageReactions

	b := &Bot{
		session: session,
		cfg:     cfg,
	}

	session.AddHandler(b.onReady)
	session.AddHandler(b.onMessageCreate)
	session.AddHandler(b.onMessageReactionAdd)

	return b, nil
}

func (b *Bot) Start() error {
	if err := b.session.Open(); err != nil {
		return err
	}

	return nil
}

func (b *Bot) Stop() {
	b.session.Close()
}

func (b *Bot) onReady(_ *discordgo.Session, event *discordgo.Ready) {
	log.Printf("logged in as %s\n", event.User.Username)
}

func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.Bot {
		return
	}

	if m.ChannelID != b.cfg.LinksChannelId {
		return
	}

	if _, err := url.ParseRequestURI(m.Content); err != nil {
		linkdingAdminUsers := strings.Split(b.cfg.LinkdingAdminUsers, ",")
		if slices.Contains(linkdingAdminUsers, m.Author.ID) {
			return
		}

		if err := s.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
			log.Printf("failed to delete message %s in channel %s: %s", m.ID, m.ChannelID, err)
			return
		}
	}
}

func (b *Bot) onMessageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.ChannelID != b.cfg.LinksChannelId {
		return
	}

	linkdingAdminUsers := strings.Split(b.cfg.LinkdingAdminUsers, ",")
	if !slices.Contains(linkdingAdminUsers, r.UserID) {
		return
	}

	msg, err := s.ChannelMessage(r.ChannelID, r.MessageID)
	if err != nil {
		log.Printf("failed to fetch message %s: %s", r.MessageID, err)
		return
	}

	if msg.Author == nil || msg.Author.Bot {
		return
	}

	if r.Emoji.Name != "🔗" {
		return
	}

	url := msg.Content
	title := ""
	description := ""
	poster := msg.Author.Username

	for _, embed := range msg.Embeds {
		if embed.URL == url {
			title = embed.Title
			description = embed.Description
			break
		}
	}

	if title == "" {
		title, err = utils.GetPageTitle(url)
		if err != nil {
			log.Printf("failed to extract title of the url: %s", url)
			return
		}
	}

	if err := helpers.AddBookmarkToLinkding(helpers.LinkdingConfig{
		BaseApiUrl: b.cfg.LinkdingBaseUrl,
		ApiToken:   b.cfg.LinkdingApiToken,
	}, url, title, description, poster); err != nil {
		log.Printf("failed to add to linkding: %s", err)
		return
	}
}
