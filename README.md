# Hytale Watcher Bot

[EN](#EN) // [RU](#RU)

## EN

Source code for an unofficial Telegram bot for checking the Hytale launcher version.

[Bot link](https://t.me/hytale_watcher_bot)

### Reference Information

1. Create a .env file next to the bot executable and fill it with the following:

```
HYTALE_WATCHER_BOT_TOKEN=TOKEN
ADMIN_ID=ID
```

Where:
- `TOKEN` is the generated token for your bot in `@BotFather`;
- `ID` is the Telegram ID of the bot administrator (or creator) for accessing bot statistics.

2. Bot settings for group interaction: enable `Allow Groups` and `Delete Messages` within `Group Admin Rights`; the `Manage Group` rule within `Group Admin Rights` will be enabled automatically, but it cannot be removed separately; it will be re-enabled.
3. Commands:
- `/start` - Select language (automatic subscription to versions updates after selecting a language);
- `/subscribe` - Subscribe to the current chat for news about the launcher and JRE updates, asks to specify the language if the chat has been unsubscribed;
- `/unsubscribe` - Unsubscribe from the launcher and JRE updates;
- `/show_launcher` - Show the launcher version and download links;
- `/show_jre` - Show the JRE version and download links;
- `/show_timer` - Show the bot timer update time;
- `/admin_stats` - Show bot statistics (number of subscribers to updates, only for the bot administrator/creator);
- `/help` - Show help.

## RU

Исходный код неофициального бота для Telegram, для проверки версии лаунчера Hytale.

[Ссылка на бота](https://t.me/hytale_watcher_bot)

### Справочная информация

1. Рядом с исполняемым файлом бота нужно создать файл `.env` и внутри заполнить его таким образом:

```
HYTALE_WATCHER_BOT_TOKEN=TOKEN
ADMIN_ID=ID
```

где:
- `TOKEN` - это сгенерированный токен вашего бота в `@BotFather`;
- `ID` - это Telegram ID администратора (или создателя) бота для доступа к статистике бота.

2. Настройки бота для взаимодействия внутри группы: разрешить `Allow Groups` и `Delete Messages` внутри `Group Admin Rights`; правило `Manage Group` внутри `Group Admin Rights` разрешится автоматически, но убрать его отдельно не получится, он разрешится вновь
3. Команды:
- `/start` - Выбрать язык (автоматическая подписка на обновления версий после выбора языка);
- `/subscribe` - Подписать текущий чат на новости об обновлении лаунчера и JRE, просит указать язык, если чат был отписан;
- `/unsubscribe` - Отписаться от обновления версии;
- `/show_launcher` - Показать версию и ссылки для загрузки лаунчера;
- `/show_jre` - Показать версию и ссылки для загрузки JRE;
- `/show_timer` - Показать время обновления таймера бота;
- `/admin_stats` - Показать статиститку бота (количество подписавшихся на обновления, только для администратора/создателя бота);
- `/help` - Показать справку.
