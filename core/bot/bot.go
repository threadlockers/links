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
var HOURGLASS_EMOJI = "⏳"

var EMOJI_TAG_MAP = config.EmojiTagMap

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
	session.AddHandler(b.onMessageReactionAddTag)
	session.AddHandler(b.onMessageReactionRemoveTag)

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

func (b *Bot) isAdmin(userID string) bool {
	admins := strings.Split(b.cfg.LinkdingAdminUsers, ",")
	return slices.Contains(admins, userID)
}

func (b *Bot) isLinksChannel(channelID string) bool {
	return channelID == b.cfg.LinksChannelId
}

func (b *Bot) addReaction(channelID, messageID, emoji string) {
	if err := b.session.MessageReactionAdd(channelID, messageID, emoji); err != nil {
		log.Printf("failed to add %s to message %s: %s", emoji, messageID, err)
	}
}

func (b *Bot) removeReaction(channelID, messageID, emoji, userID string) {
	if err := b.session.MessageReactionRemove(channelID, messageID, emoji, userID); err != nil {
		log.Printf("failed to remove %s from message %s: %s", emoji, messageID, err)
	}
}

func (b *Bot) removeBotReaction(channelID, messageID, emoji string) {
	if err := b.session.MessageReactionRemove(channelID, messageID, emoji, "@me"); err != nil {
		log.Printf("failed to remove bot %s from message %s: %s", emoji, messageID, err)
	}
}

func (b *Bot) hasReaction(msg *discordgo.Message, emoji string) bool {
	for _, r := range msg.Reactions {
		if r.Emoji.Name == emoji {
			return true
		}
	}
	return false
}

func (b *Bot) onMessageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if !b.isLinksChannel(r.ChannelID) {
		return
	}
	if !b.isAdmin(r.UserID) {
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

	if r.Emoji.Name != LINK_EMOJI {
		return
	}

	if b.hasReaction(msg, CHECKMARK_EMOJI) {
		b.removeBotReaction(r.ChannelID, r.MessageID, CHECKMARK_EMOJI)
	}
	b.addReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)

	url, remaining := utils.ExtractUrlAndRemainingText(msg.Content)
	if url == nil {
		b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
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
			b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
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
		b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
		return
	}

	b.removeReaction(r.ChannelID, r.MessageID, LINK_EMOJI, r.UserID)
	b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
	b.addReaction(r.ChannelID, r.MessageID, CHECKMARK_EMOJI)
}

func (b *Bot) onMessageReactionAddTag(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if !b.isLinksChannel(r.ChannelID) {
		return
	}
	if !b.isAdmin(r.UserID) {
		return
	}

	if r.Emoji.Name == LINK_EMOJI {
		return
	}

	tagName, ok := EMOJI_TAG_MAP[r.Emoji.Name]
	if !ok {
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

	if b.hasReaction(msg, CHECKMARK_EMOJI) {
		b.removeBotReaction(r.ChannelID, r.MessageID, CHECKMARK_EMOJI)
	}
	b.addReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)

	url, _ := utils.ExtractUrlAndRemainingText(msg.Content)
	if url == nil {
		b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
		return
	}

	linkdingCfg := helpers.LinkdingConfig{
		BaseApiUrl: b.cfg.LinkdingBaseUrl,
		ApiToken:   b.cfg.LinkdingApiToken,
	}

	bookmark, err := helpers.GetBookmarkByUrl(linkdingCfg, url.String())
	if err != nil {
		log.Printf("failed to look up bookmark: %s", err)
		b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
		return
	}
	if bookmark == nil {
		log.Printf("bookmark not found for url %s, skipping tag", url.String())
		b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
		return
	}

	mergedTags := bookmark.TagNames
	if slices.Contains(mergedTags, tagName) {
		log.Printf("bookmark %d already has tag %q, skipping", bookmark.ID, tagName)
		b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
		b.addReaction(r.ChannelID, r.MessageID, CHECKMARK_EMOJI)
		return
	}
	mergedTags = append(mergedTags, tagName)

	if err := helpers.UpdateBookmarkTags(linkdingCfg, bookmark.ID, mergedTags); err != nil {
		log.Printf("failed to update tags on bookmark %d: %s", bookmark.ID, err)
		b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
		return
	}

	log.Printf("added tag %q to bookmark %d (%s)", tagName, bookmark.ID, url.String())
	b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
	b.addReaction(r.ChannelID, r.MessageID, CHECKMARK_EMOJI)
}

func (b *Bot) onMessageReactionRemoveTag(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if !b.isLinksChannel(r.ChannelID) {
		return
	}
	if !b.isAdmin(r.UserID) {
		return
	}

	tagName, ok := EMOJI_TAG_MAP[r.Emoji.Name]
	if !ok {
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

	if b.hasReaction(msg, CHECKMARK_EMOJI) {
		b.removeBotReaction(r.ChannelID, r.MessageID, CHECKMARK_EMOJI)
	}
	b.addReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)

	url, _ := utils.ExtractUrlAndRemainingText(msg.Content)
	if url == nil {
		b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
		return
	}

	linkdingCfg := helpers.LinkdingConfig{
		BaseApiUrl: b.cfg.LinkdingBaseUrl,
		ApiToken:   b.cfg.LinkdingApiToken,
	}

	bookmark, err := helpers.GetBookmarkByUrl(linkdingCfg, url.String())
	if err != nil {
		log.Printf("failed to look up bookmark: %s", err)
		b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
		return
	}
	if bookmark == nil {
		b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
		return
	}

	if !slices.Contains(bookmark.TagNames, tagName) {
		b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
		b.addReaction(r.ChannelID, r.MessageID, CHECKMARK_EMOJI)
		return
	}
	updatedTags := slices.DeleteFunc(bookmark.TagNames, func(t string) bool {
		return t == tagName
	})

	if err := helpers.UpdateBookmarkTags(linkdingCfg, bookmark.ID, updatedTags); err != nil {
		log.Printf("failed to update tags on bookmark %d: %s", bookmark.ID, err)
		b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
		return
	}

	log.Printf("removed tag %q from bookmark %d (%s)", tagName, bookmark.ID, url.String())
	b.removeBotReaction(r.ChannelID, r.MessageID, HOURGLASS_EMOJI)
	b.addReaction(r.ChannelID, r.MessageID, CHECKMARK_EMOJI)
}
