package config

type EnvCfg struct {
	DiscordBotToken string `env:"DISCORD_BOT_TOKEN,required"`
}
