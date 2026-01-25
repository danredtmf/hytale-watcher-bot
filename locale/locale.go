package locale

type Lang string
type CheckType int
type MessageType int
type BroadcastType int

const (
	EN Lang = "en"
	RU Lang = "ru"
)

const (
	CheckLauncher CheckType = iota
	CheckJRE
)

const (
	SubscribeSuccess MessageType = iota
	UnsubscribeSuccess
	SelectLanguage
	LangSet
	NewVersionLauncher
	NewVersionJRE
	NewVersionTemplate
	Help
	BtnRussian
	BtnEnglish
	LinkInstaller
	LinkZip
	LinkTar
	ShowTimerNever
	ShowTimerUntil
	AdminStats
	AdminForceCheck
)

const (
	UpdateLauncher BroadcastType = iota
	UpdateJRE
)

var Map = map[Lang]map[MessageType]string{
	RU: {
		SubscribeSuccess:   "✅ Подписаны / Вы будете получать уведомления об обновлениях.",
		UnsubscribeSuccess: "❌ Отписаны / Уведомления больше не будут приходить.",
		SelectLanguage:     "Выберите язык:",
		LangSet:            "Язык установлен!",
		NewVersionLauncher: "лаунчера Hytale",
		NewVersionJRE:      "JRE",
		NewVersionTemplate: `🚀 Произошло обновление сигнатуры <b>%s</b>:

<b>Версия</b>: <code>%s</code> ➜ <code>%s</code>

SHA256:
<b>Windows</b>: <code>%s</code> ➜ <code>%s</code>
<b>Linux</b>: <code>%s</code> ➜ <code>%s</code>
<b>Darwin</b>: <code>%s</code> ➜ <code>%s</code>`,
		BtnRussian:    "🇷🇺 Russian",
		BtnEnglish:    "🇺🇸 English",
		LinkInstaller: "Установщик",
		LinkZip:       "ZIP",
		LinkTar:       "TAR.GZ",
		Help: `❓ <b>Как пользоваться ботом:</b>

• /start — сменить язык (автоматическая подписка на обновления версии после выбора языка);
• /subscribe — подписать текущий чат на новости об обновлении лаунчера и JRE, просит указать язык, если чат был отписан;
• /unsubscribe — отменить подписку;
• /show_launcher — показать версию и ссылки для загрузки лаунчера;
• /show_jre — показать версию и ссылки для загрузки JRE;
• /show_timer — показать время обновления таймера бота;
• /help — показать справку.

👥 <b>В группах:</b>
Просто добавьте бота в группу и сделайте его администратором (необязательно, но желательно для стабильной работы). Используйте команды с тегом бота, если включена защита от спама, например: <code>/subscribe@hytale_watcher_bot</code>.`,
		ShowTimerNever: "⏳ Таймер ещё не запущен",
		ShowTimerUntil: "⏱ Следующая проверка через %d мин %d сек",
		AdminStats: `📊 <b>Статистика подписчиков</b>
👥 Всего: <b>%d</b>
🇷🇺 RU: <b>%d</b>
🇬🇧 EN: <b>%d</b>

📊 <b>Количество проверок обновлений: %s</b>`,
		AdminForceCheck: "✅ Проверка обновлений запущена вручную",
	},
	EN: {
		SubscribeSuccess:   "✅ Subscribed / You will receive update notifications.",
		UnsubscribeSuccess: "❌ Unsubscribed / You will no longer receive notifications.",
		SelectLanguage:     "Select language:",
		LangSet:            "Language set!",
		NewVersionLauncher: "Hytale launcher",
		NewVersionJRE:      "JRE",
		NewVersionTemplate: `🚀 The <b>%s</b> signature has been updated:
<b>Version</b>: <code>%s</code> ➜ <code>%s</code>

SHA256:
<b>Windows</b>: <code>%s</code> ➜ <code>%s</code>
<b>Linux</b>: <code>%s</code> ➜ <code>%s</code>
<b>Darwin</b>: <code>%s</code> ➜ <code>%s</code>`,
		BtnRussian:    "🇷🇺 Russian",
		BtnEnglish:    "🇺🇸 English",
		LinkInstaller: "Installer",
		LinkZip:       "ZIP",
		LinkTar:       "TAR.GZ",
		Help: `❓ <b>How to use the bot:</b>

• /start — change the language (automatic subscription to version updates after selecting a language);
• /subscribe — subscribe to the current chat for news about the launcher and JRE updates, asks to specify the language if the chat has been unsubscribed;
• /unsubscribe — unsubscribe;
• /show_launcher — show the launcher version and download links;
• /show_jre — show the JRE version and download links;
• /show_timer — show the bot timer update time;
• /help — show help.

👥 <b>In groups:</b>
Simply add the bot to the group and make it an administrator (optional, but recommended for stable operation). Use commands with the bot tag if spam protection is enabled, for example: <code>/subscribe@hytale_watcher_bot</code>.`,
		ShowTimerNever: "⏳ The timer has not started yet",
		ShowTimerUntil: "⏱ Next check in %d min %d sec",
		AdminStats: `📊 <b>Subscriber statistics:</b>
👥 Total: <b>%d</b>
🇷🇺 RU: <b>%d</b>
🇬🇧 EN: <b>%d</b>

📊 <b>Number of update checks: %s</b>`,
		AdminForceCheck: "✅ Check for updates started manually",
	},
}

const HytaleLauncherLinksTemplate = `📊 <b>Hytale Launcher:</b> <code>%s</code>

SHA256:
<b>Windows</b>: <code>%s</code>
<b>Linux</b>: <code>%s</code>
<b>Darwin</b>: <code>%s</code>`

const HytaleJRELinksTemplate = `📊 <b>JRE:</b> <code>%s</code>

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
