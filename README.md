# FreeFeed Telegram Client

## Usage

### Command line

Flags of freefeed-tg-client.exe (eider -token or -token-file must be specified):

    -token string
        Telegram bot token
    -token-file string
        Path to the file with Telegram bot token
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
    -no-content
        Do not include post/comment content into the TG messages

### Docker

Set the `TOKEN` environment variable to the value of Telegram bot token. Mount
the `/bot/data` volume to the writable directory. Use `UID`/`GID` variables to
set uid/gid of the running process.

You can set the `DEBUG` environment variable to `*` to see all debug messages.

## Development

### Build

`go build [-o output_file]`

For cross-platform builds, use GOOS and GOARCH [environment variables](https://go.dev/doc/install/source#environment).

### Run (some) tests

`go test ./...`

### Text translation

⚠ For now, this feature only works with Go up to 1.17 ([issue](https://github.com/golang/go/issues/51822)).

Run `go generate`. Manually create the missing entries in `/locales/ru/messages.gotext.json`
to update text translations. Then run `go generate` again.