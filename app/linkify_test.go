package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinkify(t *testing.T) {
	assert := assert.New(t)

	a := &App{FreeFeedHost: "freefeed.net"}

	assert.Equal(
		`hello, <a href="https://freefeed.net/username">@username</a>!`,
		a.Linkify(`hello, @username!`),
	)

	assert.Equal(
		`hello, <a href="https://freefeed.net/user-name">@user-name</a>!`,
		a.Linkify(`hello, @user-name!`),
	)

	assert.Equal(
		`&lt;big&gt;hello&amp;&lt;/big&gt;, <a href="https://freefeed.net/user-name">@user-name</a>!`,
		a.Linkify(`<big>hello&</big>, @user-name!`),
	)
}
