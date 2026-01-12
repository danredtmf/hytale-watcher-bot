package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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

type HytaleVersion struct {
	Version     string `json:"version"`
	DownloadURL struct {
		Linux   PlatformArch `json:"linux"`
		Darwin  PlatformArch `json:"darwin"`
		Windows PlatformArch `json:"windows"`
	} `json:"download_url"`
}

type HytaleVersionDiff struct {
	VersionPrevious string
	SHA256Previous  string
	VersionCurrent  string
	SHA256Current   string
}

var (
	lastRequest         = make(map[int64]time.Time)
	mu                  sync.Mutex
	msgIDSelectLanguage telego.MessageID
)

func main() {
	_ = godotenv.Load()
	token := os.Getenv("HYTALE_WATCHER_BOT_TOKEN")
	if token == "" {
		log.Fatal("Токен не найден!")
	}

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

	ticker := time.NewTicker(10 * time.Minute)
	fmt.Println("Бот запущен и проверяет обновления...")

	var dataDiff HytaleVersionDiff
	checkVersion(ctx, bot, db, &dataDiff)

	for range ticker.C {
		checkVersion(ctx, bot, db, &dataDiff)
	}
}

func checkVersion(ctx context.Context, bot *telego.Bot, db *sql.DB, dataDiff *HytaleVersionDiff) {
	resp, err := http.Get("https://launcher.hytale.com/version/release/launcher.json")
	if err != nil {
		log.Printf("Network error: %v", err)
		return
	}
	defer resp.Body.Close()

	var data HytaleVersion
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("JSON error: %v", err)
		return
	}

	var lastVersion, lastCombinedHash string
	_ = db.QueryRow("SELECT value FROM settings WHERE key = 'last_version'").Scan(&lastVersion)
	_ = db.QueryRow("SELECT value FROM settings WHERE key = 'last_combined_hash'").Scan(&lastCombinedHash)

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

		_, _ = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('last_version', ?)", data.Version)
		_, _ = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('last_combined_hash', ?)", currentCombinedHash)

		sendBroadcast(ctx, bot, db, dataDiff)
	}
}

func sendBroadcast(ctx context.Context, bot *telego.Bot, db *sql.DB, dataDiff *HytaleVersionDiff) {
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

	wP, lP, dP := parseHashes(dataDiff.SHA256Previous)
	wC, lC, dC := parseHashes(dataDiff.SHA256Current)

	for rows.Next() {
		var id int64
		var lang locale.Lang
		if err := rows.Scan(&id, &lang); err != nil {
			continue
		}

		msg := fmt.Sprintf(locale.Get(lang, locale.NewVersion),
			dataDiff.VersionPrevious, dataDiff.VersionCurrent,
			wP, wC, lP, lC, dP, dC,
		)
		_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(id), msg).WithParseMode(telego.ModeHTML))
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
		_ = db.QueryRow("SELECT lang_key FROM subscribers WHERE user_id = ?", chatID).Scan(&lang)

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
			keyboard := tu.InlineKeyboard(
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton(locale.Get(lang, locale.BtnRussian)).WithCallbackData("lang_ru"),
					tu.InlineKeyboardButton(locale.Get(lang, locale.BtnEnglish)).WithCallbackData("lang_en"),
				),
			)
			resultMsg, err := bot.SendMessage(ctx, tu.Message(tu.ID(chatID), locale.Get(lang, locale.SelectLanguage)).WithReplyMarkup(keyboard))
			if err == nil {
				msgIDSelectLanguage.MessageID = resultMsg.MessageID
			}

			continue
		}

		if strings.HasPrefix(text, "/subscribe") {
			_, err := db.Exec("INSERT OR IGNORE INTO subscribers (user_id, lang_key) VALUES (?, ?)", chatID, lang)
			if err == nil {
				_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(chatID), locale.Get(lang, locale.SubscribeSuccess)))
			}
			continue
		}

		if strings.HasPrefix(text, "/unsubscribe") {
			_, _ = db.Exec("DELETE FROM subscribers WHERE user_id = ?", chatID)
			_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(chatID), locale.Get(lang, locale.UnsubscribeSuccess)))
			continue
		}

		if strings.HasPrefix(text, "/show") {
			showLinks(ctx, bot, db, chatID, lang)
			continue
		}

		if strings.HasPrefix(text, "/help") {
			_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(chatID), locale.Get(lang, locale.Help)).WithParseMode(telego.ModeHTML))
		}
	}
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
	}
}

func showLinks(ctx context.Context, bot *telego.Bot, db *sql.DB, chatID int64, lang locale.Lang) {
	var ver, sha string
	_ = db.QueryRow("SELECT value FROM settings WHERE key = 'last_version'").Scan(&ver)
	_ = db.QueryRow("SELECT value FROM settings WHERE key = 'last_combined_hash'").Scan(&sha)

	if ver == "" {
		ver = "unknown"
	}

	w, l, d := "n/a", "n/a", "n/a"
	arr := strings.Split(sha, ",")
	if len(arr) >= 3 {
		w, l, d = arr[0], arr[1], arr[2]
	}

	text := fmt.Sprintf(locale.HytaleLinksTemplate, ver, w, l, d)
	latestText := locale.Get(lang, locale.LatestLink)
	currentText := locale.Get(lang, locale.CurrentLink)

	keyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnWindows, latestText)).WithURL("https://launcher.hytale.com/builds/release/windows/amd64/hytale-launcher-installer-latest.exe"),
			tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnDarwin, latestText)).WithURL("https://launcher.hytale.com/builds/release/darwin/arm64/hytale-launcher-latest.dmg"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnLinuxFlatpak, latestText)).WithURL("https://launcher.hytale.com/builds/release/linux/amd64/hytale-launcher-latest.flatpak"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnWindows, currentText)).WithURL(fmt.Sprintf("https://launcher.hytale.com/builds/release/windows/amd64/hytale-launcher-%s.zip", ver)),
			tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnDarwin, currentText)).WithURL(fmt.Sprintf("https://launcher.hytale.com/builds/release/darwin/arm64/hytale-launcher-%s.zip", ver)),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(fmt.Sprintf(locale.BtnLinux, currentText)).WithURL(fmt.Sprintf("https://launcher.hytale.com/builds/release/linux/amd64/hytale-launcher-%s.zip", ver)),
		),
	)

	_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(chatID), text).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard))
}
