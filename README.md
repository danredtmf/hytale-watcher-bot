# Hytale Watcher Bot

[EN](#EN) // [RU](#RU)

## EN

Source code for an unofficial Telegram bot for checking the Hytale launcher version.

[Bot link](https://t.me/hytale_watcher_bot)

### Reference Information

1. Create a .env file next to the bot executable and fill it with the following:

```
HYTALE_WATCHER_BOT_TOKEN=TOKEN
```

where `TOKEN` is the generated token for your bot in `@BotFather`

2. Bot settings for group interaction: enable `Allow Groups` and `Delete Messages` within `Group Admin Rights`; the `Manage Group` rule within `Group Admin Rights` will be enabled automatically, but it cannot be removed separately; it will be re-enabled.
3. Commands:
- `/start` - Select language (automatic subscription to version updates after selecting a language);
- `/subscribe` - Subscribe to version updates;
- `/unsubscribe` - Unsubscribe from version updates;
- `/show` - Show the version and download links for the launcher;
- `/help` - Show help.

## RU

Исходный код неофициального бота для Telegram, для проверки версии лаунчера Hytale.

[Ссылка на бота](https://t.me/hytale_watcher_bot)

### Справочная информация

1. Рядом с исполняемым файлом бота нужно создать файл `.env` и внутри заполнить его таким образом:

```
HYTALE_WATCHER_BOT_TOKEN=TOKEN
```

где `TOKEN` - это сгенерированный токен вашего бота в `@BotFather`

2. Настройки бота для взаимодействия внутри группы: разрешить `Allow Groups` и `Delete Messages` внутри `Group Admin Rights`; правило `Manage Group` внутри `Group Admin Rights` разрешится автоматически, но убрать его отдельно не получится, он разрешится вновь
3. Команды:
- `/start` - Выбрать язык (автоматическая подписка на обновления версии после выбора языка);
- `/subscribe` - Подписаться на обновление версии;
- `/unsubscribe` - Отписаться от обновления версии;
- `/show` - Показать версию и ссылки для загрузки лаунчера;
- `/help` - Показать справку.
