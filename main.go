package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/danredtmf/hytale-watcher-bot/locale"
	"github.com/joho/godotenv"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	_ "modernc.org/sqlite"
)

type Artifact struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type PlatformArch struct {
	AMD64 *Artifact `json:"amd64,omitempty"`
	ARM64 *Artifact `json:"arm64,omitempty"`
}

type Version struct {
	Version     string `json:"version"`
	DownloadURL struct {
		Linux   PlatformArch `json:"linux"`
		Darwin  PlatformArch `json:"darwin"`
		Windows PlatformArch `json:"windows"`
	} `json:"download_url"`
}

type VersionDiff struct {
	VersionPrevious string
	SHA256Previous  string
	VersionCurrent  string
	SHA256Current   string
}

type SettingsNames struct {
	Keys [2]string
}

const (
	LauncherVersionLink = "https://launcher.hytale.com/version/release/launcher.json"
	JREVersionLink      = "https://launcher.hytale.com/version/release/jre.json"

	WindowsLauncherLinkLatest = "https://launcher.hytale.com/builds/release/windows/amd64/hytale-launcher-installer-latest.exe"
	DarwinLauncherLinkLatest  = "https://launcher.hytale.com/builds/release/darwin/arm64/hytale-launcher-latest.dmg"
	LinuxLauncherLinkLatest   = "https://launcher.hytale.com/builds/release/linux/amd64/hytale-launcher-latest.flatpak"

	WindowsLauncherLinkCurrent = "https://launcher.hytale.com/builds/release/windows/amd64/hytale-launcher-%s.zip"
	DarwinLauncherLinkCurrent  = "https://launcher.hytale.com/builds/release/darwin/arm64/hytale-launcher-%s.zip"
	LinuxLauncherLinkCurrent   = "https://launcher.hytale.com/builds/release/linux/amd64/hytale-launcher-%s.zip"

	WindowsJRELinkCurrent = "https://launcher.hytale.com/redist/jre/linux/amd64/jre-%s.tar.gz"
	DarwinJRELinkCurrent  = "https://launcher.hytale.com/redist/jre/darwin/arm64/jre-%s.tar.gz"
	LinuxJRELinkCurrent   = "https://launcher.hytale.com/redist/jre/windows/amd64/jre-%s.zip"

	DBNextCheckTimeKey = "next_check_time"
	DBCheckCounterKey  = "check_counter"

	DBSelectSettingsKey       = "SELECT value FROM settings WHERE key = ?"
	DBInsertSettingsKey       = "INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)"
	DBInsertOrIgnoreSubscribe = "INSERT OR IGNORE INTO subscribers (user_id, lang_key) VALUES (?, ?)"
)

var (
	lastRequest         = make(map[int64]time.Time)
	mu                  sync.Mutex
	msgIDSelectLanguage telego.MessageID
	adminID             int64
	lastCheckTime       time.Time
	nextCheckTime       time.Time
)

func main() {
	_ = godotenv.Load()
	token := os.Getenv("HYTALE_WATCHER_BOT_TOKEN")
	if token == "" {
		log.Fatal("Токен не найден!")
	}

	adminIDConv, err := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)
	if err != nil {
		log.Fatal("Что-то с парсингом токена!")
	}
	adminID = adminIDConv
	adminIDConv = 0

	db, err := sql.Open("sqlite", "hytale_watcher_bot.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Оптимизация SQLite (WAL режим позволяет читать во время записи)
	_, _ = db.Exec(`PRAGMA journal_mode=WAL;`)
	_, _ = db.Exec(`PRAGMA synchronous=NORMAL;`)

	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS subscribers (user_id INTEGER PRIMARY KEY, lang_key TEXT DEFAULT 'en');`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT);`)

	nextCheckTime = loadNextCheckTime(db)

	if nextCheckTime.IsZero() || nextCheckTime.Before(time.Now()) {
		interval := randomInterval()
		nextCheckTime = time.Now().Add(interval)
		saveNextCheckTime(db, nextCheckTime)
	}

	bot, err := telego.NewBot(token,
		telego.WithHTTPClient(&http.Client{
			Timeout: time.Second * 30,
		}),
		telego.WithDefaultLogger(false, true),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	updates, _ := bot.UpdatesViaLongPolling(ctx, nil)

	go handleUpdates(ctx, bot, db, updates)

	fmt.Println("Бот запущен и проверяет обновления...")

	var dataLauncherDiff VersionDiff
	var dataJREDiff VersionDiff

	launcherNames := SettingsNames{Keys: [2]string{"last_version", "last_combined_hash"}}
	jreNames := SettingsNames{Keys: [2]string{"last_version_jre", "last_combined_hash_jre"}}

	lastCheckTime = time.Now()
	check(ctx, bot, db,
		[2]*VersionDiff{&dataLauncherDiff, &dataJREDiff},
		[2]SettingsNames{launcherNames, jreNames},
	)

	go func() {
		for {
			sleep := time.Until(nextCheckTime)
			sleep = max(sleep, time.Duration(0))
			time.Sleep(sleep)

			lastCheckTime = time.Now()

			check(ctx, bot, db,
				[2]*VersionDiff{&dataLauncherDiff, &dataJREDiff},
				[2]SettingsNames{launcherNames, jreNames},
			)

			interval := randomInterval()
			nextCheckTime = time.Now().Add(interval)
			saveNextCheckTime(db, nextCheckTime)
		}
	}()

	select {}
}

func randomInterval() time.Duration {
	return time.Duration(3+rand.Intn(3)) * time.Minute
}

func check(ctx context.Context, bot *telego.Bot, db *sql.DB, dataDiffs [2]*VersionDiff, settingsNamesArr [2]SettingsNames) {
	var checks [2]bool

	checks[0] = checkVersion(db, locale.CheckLauncher, dataDiffs[0], settingsNamesArr[0])
	checks[1] = checkVersion(db, locale.CheckJRE, dataDiffs[1], settingsNamesArr[1])

	sendBroadcast(ctx, bot, db, dataDiffs, checks)

	incrementCheckCounter(db)
}

func checkVersion(db *sql.DB, checkType locale.CheckType, dataDiff *VersionDiff, settingsNames SettingsNames) (isNewVersionAvailable bool) {
	var versionLink string

	switch checkType {
	case locale.CheckLauncher:
		versionLink = LauncherVersionLink
	case locale.CheckJRE:
		versionLink = JREVersionLink
	}

	resp, err := http.Get(versionLink)
	if err != nil {
		log.Printf("Network error: %v", err)
		return
	}
	defer resp.Body.Close()

	var data Version
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("JSON error: %v", err)
		return
	}

	var lastVersion, lastCombinedHash string
	_ = db.QueryRow(DBSelectSettingsKey, settingsNames.Keys[0]).Scan(&lastVersion)
	_ = db.QueryRow(DBSelectSettingsKey, settingsNames.Keys[1]).Scan(&lastCombinedHash)

	hashes := make([]string, 3)
	hashes[0], hashes[1], hashes[2] = "n/a", "n/a", "n/a"

	if data.DownloadURL.Windows.AMD64 != nil {
		hashes[0] = data.DownloadURL.Windows.AMD64.SHA256
	}
	if data.DownloadURL.Linux.AMD64 != nil {
		hashes[1] = data.DownloadURL.Linux.AMD64.SHA256
	}
	if data.DownloadURL.Darwin.ARM64 != nil {
		hashes[2] = data.DownloadURL.Darwin.ARM64.SHA256
	}

	currentCombinedHash := strings.Join(hashes, ",")

	if data.Version != lastVersion || currentCombinedHash != lastCombinedHash {
		dataDiff.VersionPrevious = lastVersion
		dataDiff.VersionCurrent = data.Version
		dataDiff.SHA256Previous = lastCombinedHash
		dataDiff.SHA256Current = currentCombinedHash

		_, _ = db.Exec(DBInsertSettingsKey, settingsNames.Keys[0], data.Version)
		_, _ = db.Exec(DBInsertSettingsKey, settingsNames.Keys[1], currentCombinedHash)

		return true
	}

	return false
}

func sendBroadcast(ctx context.Context, bot *telego.Bot, db *sql.DB, dataDiffs [2]*VersionDiff, checks [2]bool) {
	rows, err := db.Query("SELECT user_id, lang_key FROM subscribers")
	if err != nil {
		log.Printf("Broadcast DB error: %v", err)
		return
	}
	defer rows.Close()

	// Вспомогательная функция для парсинга хешей
	parseHashes := func(combined string) (w, l, d string) {
		w, l, d = "n/a", "n/a", "n/a"
		arr := strings.Split(combined, ",")
		if len(arr) >= 3 {
			return arr[0], arr[1], arr[2]
		}
		return
	}

	// Launcher SHA256
	wPL, lPL, dPL := parseHashes(dataDiffs[0].SHA256Previous)
	wCL, lCL, dCL := parseHashes(dataDiffs[0].SHA256Current)

	// JRE SHA256
	wPJRE, lPJRE, dPJRE := parseHashes(dataDiffs[1].SHA256Previous)
	wCJRE, lCJRE, dCJRE := parseHashes(dataDiffs[1].SHA256Current)

	for rows.Next() {
		var id int64
		var lang locale.Lang
		if err := rows.Scan(&id, &lang); err != nil {
			continue
		}

		// Launcher
		if checks[0] {
			msgLauncher := fmt.Sprintf(locale.Get(lang, locale.NewVersionTemplate),
				locale.Get(lang, locale.NewVersionLauncher), dataDiffs[0].VersionPrevious, dataDiffs[0].VersionCurrent,
				wPL, wCL, lPL, lCL, dPL, dCL,
			)

			_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(id), msgLauncher).WithParseMode(telego.ModeHTML))
		}

		// JRE
		if checks[1] {
			msgJRE := fmt.Sprintf(locale.Get(lang, locale.NewVersionTemplate),
				locale.Get(lang, locale.NewVersionJRE), dataDiffs[1].VersionPrevious, dataDiffs[1].VersionCurrent,
				wPJRE, wCJRE, lPJRE, lCJRE, dPJRE, dCJRE,
			)

			_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(id), msgJRE).WithParseMode(telego.ModeHTML))
		}
	}
}

func handleUpdates(ctx context.Context, bot *telego.Bot, db *sql.DB, updates <-chan telego.Update) {
	for update := range updates {
		if update.CallbackQuery != nil {
			handleCallback(ctx, bot, db, update.CallbackQuery)
			continue
		}
		if update.Message == nil {
			continue
		}

		text := update.Message.Text
		chatID := update.Message.Chat.ID

		// Безопасное получение языка
		var lang locale.Lang = locale.EN
		err_usr := db.QueryRow("SELECT lang_key FROM subscribers WHERE user_id = ?", chatID).Scan(&lang)

		// Rate Limiting с мьютексом
		mu.Lock()
		last, ok := lastRequest[chatID]
		if ok && time.Since(last) < 1*time.Second {
			mu.Unlock()
			continue
		}
		lastRequest[chatID] = time.Now()
		mu.Unlock()

		if strings.HasPrefix(text, "/start") {
			startCommand(ctx, bot, chatID, lang)

			continue
		}

		if strings.HasPrefix(text, "/subscribe") {
			if getUserExists(err_usr) == false {
				startCommand(ctx, bot, chatID, lang)
			} else {
				subscribeCommand(ctx, bot, db, chatID, lang)
			}

			continue
		}

		if strings.HasPrefix(text, "/unsubscribe") {
			_, _ = db.Exec("DELETE FROM subscribers WHERE user_id = ?", chatID)
			_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(chatID), locale.Get(lang, locale.UnsubscribeSuccess)))
			continue
		}

		if strings.HasPrefix(text, "/show_launcher") {
			showLinks(ctx, bot, db, locale.CheckLauncher, chatID, lang)
			continue
		}

		if strings.HasPrefix(text, "/show_jre") {
			showLinks(ctx, bot, db, locale.CheckJRE, chatID, lang)
			continue
		}

		if strings.HasPrefix(text, "/show_timer") {
			showTimer(ctx, bot, chatID, lang)
			continue
		}

		if strings.HasPrefix(text, "/admin_stats") {
			adminStats(ctx, bot, db, chatID, lang)
			continue
		}

		if strings.HasPrefix(text, "/admin_force_check") {
			adminForceCheck(ctx, bot, db, chatID, lang)
			continue
		}

		if strings.HasPrefix(text, "/help") {
			_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(chatID), locale.Get(lang, locale.Help)).WithParseMode(telego.ModeHTML))
		}
	}
}

func getUserExists(value error) bool {
	if value == sql.ErrNoRows {
		return false
	}

	return true
}

func startCommand(ctx context.Context, bot *telego.Bot, chatID int64, lang locale.Lang) {
	keyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(locale.Get(lang, locale.BtnRussian)).WithCallbackData("lang_ru"),
			tu.InlineKeyboardButton(locale.Get(lang, locale.BtnEnglish)).WithCallbackData("lang_en"),
		),
	)
	resultMsg, err := bot.SendMessage(ctx,
		tu.Message(tu.ID(chatID), locale.Get(lang, locale.SelectLanguage)).WithReplyMarkup(keyboard),
	)
	if err == nil {
		msgIDSelectLanguage.MessageID = resultMsg.MessageID
	}
}

func subscribeCommand(ctx context.Context, bot *telego.Bot, db *sql.DB, chatID int64, lang locale.Lang) {
	_, err := db.Exec(DBInsertOrIgnoreSubscribe, chatID, lang)
	if err == nil {
		printSubscribeCommand(ctx, bot, chatID, lang)
	}
}

func printSubscribeCommand(ctx context.Context, bot *telego.Bot, chatID int64, lang locale.Lang) {
	_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(chatID), locale.Get(lang, locale.SubscribeSuccess)))
}

func handleCallback(ctx context.Context, bot *telego.Bot, db *sql.DB, query *telego.CallbackQuery) {
	msg, ok := query.Message.(*telego.Message)
	if !ok {
		return
	}
	chatID := msg.Chat.ID

	lang := locale.EN
	if query.Data == "lang_ru" {
		lang = locale.RU
	}

	_, _ = db.Exec(`INSERT INTO subscribers (user_id, lang_key) VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET lang_key=?`, chatID, lang, lang)

	_ = bot.AnswerCallbackQuery(ctx, tu.CallbackQuery(query.ID).WithText(locale.Get(lang, locale.LangSet)))

	botMsg := locale.Get(lang, locale.LangSet)
	_, err := bot.SendMessage(ctx, tu.Message(tu.ID(chatID), botMsg))
	if err == nil {
		bot.DeleteMessage(ctx, tu.Delete(tu.ID(chatID), msgIDSelectLanguage.MessageID))
		printSubscribeCommand(ctx, bot, chatID, lang)
	}
}

func showLinks(ctx context.Context, bot *telego.Bot, db *sql.DB,
	checkType locale.CheckType, chatID int64, lang locale.Lang,
) {
	var ver, sha string

	switch checkType {
	case locale.CheckLauncher:
		_ = db.QueryRow("SELECT value FROM settings WHERE key = 'last_version'").Scan(&ver)
		_ = db.QueryRow("SELECT value FROM settings WHERE key = 'last_combined_hash'").Scan(&sha)
	case locale.CheckJRE:
		_ = db.QueryRow("SELECT value FROM settings WHERE key = 'last_version_jre'").Scan(&ver)
		_ = db.QueryRow("SELECT value FROM settings WHERE key = 'last_combined_hash_jre'").Scan(&sha)
	}

	if ver == "" {
		ver = "unknown"
	}

	w, l, d := "n/a", "n/a", "n/a"
	arr := strings.Split(sha, ",")
	if len(arr) >= 3 {
		w, l, d = arr[0], arr[1], arr[2]
	}

	var text string

	switch checkType {
	case locale.CheckLauncher:
		text = fmt.Sprintf(locale.HytaleLauncherLinksTemplate, ver, w, l, d)
	case locale.CheckJRE:
		text = fmt.Sprintf(locale.HytaleJRELinksTemplate, ver, w, l, d)
	}

	installerText := locale.Get(lang, locale.LinkInstaller)
	zipText := locale.Get(lang, locale.LinkZip)
	tarText := locale.Get(lang, locale.LinkTar)

	var keyboard *telego.InlineKeyboardMarkup

	switch checkType {
	case locale.CheckLauncher:
		keyboard = tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnWindows, installerText)).WithURL(WindowsLauncherLinkLatest),
				tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnDarwin, installerText)).WithURL(DarwinLauncherLinkLatest),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnLinuxFlatpak, installerText)).WithURL(LinuxLauncherLinkLatest),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnWindows, zipText)).WithURL(fmt.Sprintf(WindowsLauncherLinkCurrent, ver)),
				tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnDarwin, zipText)).WithURL(fmt.Sprintf(DarwinLauncherLinkCurrent, ver)),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnLinux, zipText)).WithURL(fmt.Sprintf(LinuxLauncherLinkCurrent, ver)),
			),
		)
	case locale.CheckJRE:
		keyboard = tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnWindows, zipText)).WithURL(
					fmt.Sprintf(WindowsJRELinkCurrent, ver),
				),
				tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnDarwin, tarText)).WithURL(
					fmt.Sprintf(DarwinJRELinkCurrent, ver),
				),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnLinux, tarText)).WithURL(
					fmt.Sprintf(LinuxJRELinkCurrent, ver),
				),
			),
		)
	}

	_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(chatID), text).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard))
}

func showTimer(ctx context.Context, bot *telego.Bot, chatID int64, lang locale.Lang) {
	if nextCheckTime.IsZero() {
		_, _ = bot.SendMessage(ctx,
			tu.Message(tu.ID(chatID), locale.Get(lang, locale.ShowTimerNever)),
		)
		return
	}

	remaining := time.Until(nextCheckTime)
	remaining = max(remaining, time.Duration(0))

	minutes := int(remaining.Minutes())
	seconds := int(remaining.Seconds()) % 60

	text := fmt.Sprintf(
		locale.Get(lang, locale.ShowTimerUntil),
		minutes, seconds,
	)

	_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(chatID), text))
}

func adminStats(ctx context.Context, bot *telego.Bot, db *sql.DB, chatID int64, lang locale.Lang) {
	if chatID != adminID {
		return
	}

	var total, ru, en int
	var checkCounter string

	_ = db.QueryRow("SELECT COUNT(*) FROM subscribers").Scan(&total)
	_ = db.QueryRow("SELECT COUNT(*) FROM subscribers WHERE lang_key = 'ru'").Scan(&ru)
	_ = db.QueryRow("SELECT COUNT(*) FROM subscribers WHERE lang_key = 'en'").Scan(&en)

	// Извлекаем значение счетчика из таблицы настроек
	err := db.QueryRow(DBSelectSettingsKey, DBCheckCounterKey).Scan(&checkCounter)
	if err != nil || checkCounter == "" {
		checkCounter = "0"
	}

	text := fmt.Sprintf(
		locale.Get(lang, locale.AdminStats),
		total, ru, en, checkCounter,
	)

	_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(chatID), text).WithParseMode(telego.ModeHTML))
}

func adminForceCheck(ctx context.Context, bot *telego.Bot, db *sql.DB, chatID int64, lang locale.Lang) {
	if chatID != adminID {
		return
	}

	go func() {
		lastCheckTime = time.Now()

		check(ctx, bot, db,
			[2]*VersionDiff{&VersionDiff{}, &VersionDiff{}},
			[2]SettingsNames{
				{Keys: [2]string{"last_version", "last_combined_hash"}},
				{Keys: [2]string{"last_version_jre", "last_combined_hash_jre"}},
			},
		)

		interval := randomInterval()
		nextCheckTime = time.Now().Add(interval)
		saveNextCheckTime(db, nextCheckTime)
	}()

	_, _ = bot.SendMessage(ctx,
		tu.Message(tu.ID(chatID), locale.Get(lang, locale.AdminForceCheck)),
	)
}

func saveNextCheckTime(db *sql.DB, t time.Time) {
	_, _ = db.Exec(DBInsertSettingsKey, DBNextCheckTimeKey, strconv.FormatInt(t.Unix(), 10))
}

func loadNextCheckTime(db *sql.DB) time.Time {
	var v string
	err := db.QueryRow(DBSelectSettingsKey, DBNextCheckTimeKey).Scan(&v)

	if err != nil {
		return time.Time{}
	}

	sec, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return time.Time{}
	}

	return time.Unix(sec, 0)
}

func incrementCheckCounter(db *sql.DB) {
	// Этот запрос создаст запись со значением 1, если её нет,
	// или увеличит существующее значение на 1.
	_, _ = db.Exec(`
        INSERT INTO settings (key, value) VALUES (?, '1')
        ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)
    `, DBCheckCounterKey)
}
