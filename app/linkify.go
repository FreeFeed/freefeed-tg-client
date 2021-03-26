package app

import (
	"fmt"
	"regexp"
	"strings"
)

var mentionsRe = regexp.MustCompile(`@[a-z0-9-]+`)

// Telegram understands only these entities
var htmlEscaper = strings.NewReplacer(
	`&`, "&amp;",
	`<`, "&lt;",
	`>`, "&gt;",
)

func (a *App) Linkify(text string) string {
	var result strings.Builder

	founds := mentionsRe.FindAllStringSubmatchIndex(text, -1)
	prev := 0
	for _, loc := range founds {
		result.WriteString(htmlEscaper.Replace(text[prev:loc[0]]))
		username := text[loc[0]+1 : loc[1]]
		result.WriteString(fmt.Sprintf(
			`<a href="https://%s/%s">@%s</a>`,
			a.FreeFeedHost,
			username,
			username,
		))
		prev = loc[1]
	}

	result.WriteString(htmlEscaper.Replace(text[prev:]))
	return result.String()
}
