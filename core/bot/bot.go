package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/threadlockers/links/core/config"
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

	session.Identify.Intents = discordgo.IntentGuildMessages

	b := &Bot{
		session: session,
		cfg:     cfg,
	}

	session.AddHandler(b.onReady)

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
