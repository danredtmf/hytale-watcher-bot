package locale

type Lang string
type MessageType int

const (
	EN Lang = "en"
	RU Lang = "ru"
)

const (
	SubscribeSuccess MessageType = iota
	UnsubscribeSuccess
	SelectLanguage
	LangSet
	NewVersion
	Help
	BtnRussian
	BtnEnglish
	LatestLink
	CurrentLink
)

var Map = map[Lang]map[MessageType]string{
	RU: {
		SubscribeSuccess:   "✅ Подписаны / Вы будете получать уведомления об обновлениях.",
		UnsubscribeSuccess: "❌ Отписаны / Уведомления больше не будут приходить.",
		SelectLanguage:     "Выберите язык:",
		LangSet:            "Язык установлен!",
		NewVersion: `🚀 Произошло обновление сигнатуры лаунчера Hytale:

<b>Версия</b>: <code>%s</code> -> <code>%s</code>

SHA256:
<b>Windows</b>: <code>%s</code> -> <code>%s</code>
<b>Linux</b>: <code>%s</code> -> <code>%s</code>
<b>Darwin</b>: <code>%s</code> -> <code>%s</code>`,
		BtnRussian:  "🇷🇺 Russian",
		BtnEnglish:  "🇺🇸 English",
		LatestLink:  "Установщик",
		CurrentLink: "ZIP",
		Help: `❓ <b>Как пользоваться ботом:</b>

• /subscribe — подписать текущий чат на новости об обновлении лаунчера;
• /unsubscribe — отменить подписку;
• /show — показать текущую версию и ссылки на загрузку;
• /start — сменить язык (автоматическая подписка на обновления версии);
• /help — Показать справку.

👥 <b>В группах:</b>
Просто добавьте бота в группу и сделайте его администратором (необязательно, но желательно для стабильной работы). Используйте команды с тегом бота, если включена защита от спама, например: <code>/subscribe@hytale_watcher_bot</code>.`,
	},
	EN: {
		SubscribeSuccess:   "✅ Subscribed / You will receive update notifications.",
		UnsubscribeSuccess: "❌ Unsubscribed / You will no longer receive notifications.",
		SelectLanguage:     "Select language:",
		LangSet:            "Language set!",
		NewVersion: `🚀 The Hytale launcher signature has been updated:
<b>Version</b>: <code>%s</code> -> <code>%s</code>

SHA256:
<b>Windows</b>: <code>%s</code> -> <code>%s</code>
<b>Linux</b>: <code>%s</code> -> <code>%s</code>
<b>Darwin</b>: <code>%s</code> -> <code>%s</code>`,
		BtnRussian:  "🇷🇺 Russian",
		BtnEnglish:  "🇺🇸 English",
		LatestLink:  "Installer",
		CurrentLink: "ZIP",
		Help: `❓ <b>How to use the bot:</b>

• /subscribe — subscribe the current chat to news about launcher updates;
• /unsubscribe — unsubscribe;
• /show — show the current version and download links;
• /start — change the language (automatic subscription to version updates);
• /help — Show help.

👥 <b>In groups:</b>
Simply add the bot to the group and make it an administrator (optional, but recommended for stable operation). Use commands with the bot tag if spam protection is enabled, for example: <code>/subscribe@hytale_watcher_bot</code>.`,
	},
}

const HytaleLinksTemplate = `📊 <b>Hytale Launcher:</b> <code>%s</code>

SHA256:
<b>Windows</b>: <code>%s</code>
<b>Linux</b>: <code>%s</code>
<b>Darwin</b>: <code>%s</code>`

const (
	BtnWindows      = "Windows (%s)"
	BtnDarwin       = "macOS (%s)"
	BtnLinuxFlatpak = "Linux (Flatpak, %s)"
	BtnLinux        = "Linux (%s)"
)

func Get(l Lang, m MessageType) string {
	if text, ok := Map[l][m]; ok {
		return text
	}
	return Map[EN][m] // Fallback на английский
}
