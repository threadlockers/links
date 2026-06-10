package bot

import (
	"log"
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

var TWITTER_HOSTS = []string{"x.com", "twitter.com", "www.x.com", "www.twitter.com"}
var LINK_EMOJI = "🔗"
var CHECKMARK_EMOJI = "✅"

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

	url, remaining := utils.ExtractUrlAndRemainingText(msg.Content)
	if url == nil {
		return
	}

	// convert arxiv pdf links to abstract links to properly fetch title
	if url.Host == "arxiv.org" && strings.Contains(url.Path, "/pdf/") {
		url.Path = strings.Replace(url.Path, "/pdf/", "/abs/", 1)
	}

	title := ""
	description := ""
	poster := msg.Author.Username

	if slices.Contains(TWITTER_HOSTS, url.Host) {
		title, description, err = utils.GetTitleAndDescriptionForTweet(url)
		if err != nil {
			log.Printf("failed to extract tweet info from fxtwitter: %s", err)
			return
		}
	} else {
		for _, embed := range msg.Embeds {
			if embed.URL == url.String() {
				title = embed.Title
				description = embed.Description
				break
			}
		}
	}

	if title == "" {
		title, err = utils.GetPageTitle(url.String())
		if err != nil || title == "" {
			log.Printf("failed to extract title of the url, falling back to url: %s", url)
			title = url.String()
		}
	}

	if err := helpers.AddBookmarkToLinkding(helpers.LinkdingConfig{
		BaseApiUrl: b.cfg.LinkdingBaseUrl,
		ApiToken:   b.cfg.LinkdingApiToken,
	}, url.String(), title, description, poster, remaining); err != nil {
		log.Printf("failed to add to linkding: %s", err)
		return
	}

	if err := b.session.MessageReactionRemove(r.ChannelID, r.MessageID, LINK_EMOJI, r.UserID); err != nil {
		log.Printf("failed to remove link emoji from message %s: %s", r.MessageID, err)
		return
	}

	if err := b.session.MessageReactionAdd(r.ChannelID, r.MessageID, CHECKMARK_EMOJI); err != nil {
		log.Printf("failed to add checkmark emoji to message %s: %s", r.MessageID, err)
		return
	}
}
