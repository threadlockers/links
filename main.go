package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/threadlockers/links/core/bot"
	"github.com/threadlockers/links/core/config"
	"github.com/threadlockers/links/core/utils"
)

type EnvCfg struct {
	DiscordBotToken string `env:"DISCORD_BOT_TOKEN,required"`
	LinksChannelId  string `env:"LINKS_CHANNEL_ID,required"`
}

func main() {
	cfg, err := utils.ParseEnv[config.EnvCfg](".env")
	if err != nil {
		panic(fmt.Sprintf("failed to parse env: %s", err))
	}

	bot, err := bot.New(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create bot instance: %s", err))
	}

	if err := bot.Start(); err != nil {
		panic(fmt.Sprintf("failed to start the bot: %s", err))
	}
	defer bot.Stop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	log.Println("shutting down...")
}
