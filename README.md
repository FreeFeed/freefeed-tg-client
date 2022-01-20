# FreeFeed Telegram Client

## Usage

Flags of freefeed-tg-client.exe:

    -token string
        Telegram bot token (required)
    -data string
        Data directory (must be writable)
        (default "data")
    -debug string
        Debug sources, set to '*' to see all messages
    -host string
        FreeFeed API/frontend hostname
        (default "freefeed.net")
    -ua string
        User-Agent for backend requests
        (default "FreeFeedTelegramClient/1.0 (https://github.com/davidmz/freefeed-tg-client)")        

## Development



### Text translation

Run `go generate`. Manually create the missing entries in `/locales/ru/messages.gotext.json`
to update text translations. Then run `go generate` again.