package config

type EnvCfg struct {
	DiscordBotToken    string `env:"DISCORD_BOT_TOKEN,required"`
	LinksChannelId     string `env:"LINKS_CHANNEL_ID,required"`
	LinkdingBaseUrl    string `env:"LINKDING_BASE_URL,required"`
	LinkdingApiToken   string `env:"LINKDING_API_TOKEN,required"`
	LinkdingAdminUsers string `env:"LINKDING_ADMIN_USERS,required"`
}
