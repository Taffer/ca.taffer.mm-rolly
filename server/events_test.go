package main

import (
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// -----------------------------------------------------------------------------
// Tests - Mattermost Plugin Interface
// -----------------------------------------------------------------------------

// TestPluginLifecycle - Make sure OnActivate()/OnDeactivate() don't crash.
func TestPluginLifecycle(t *testing.T) {
	p := initTestPlugin(t)

	assert.Nil(t, p.OnActivate())
	assert.True(t, p.active)

	// TODO : Try ExecuteCommand while deactivated?

	assert.Nil(t, p.OnDeactivate())
	assert.False(t, p.active)
}

// TestServeHTTP - Test the HTTP response.
func TestServeHTTP(t *testing.T) {
	assert := assert.New(t)
	plugin := RollyPlugin{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	plugin.ServeHTTP(nil, w, r)

	result := w.Result()
	assert.NotNil(result)
	bodyBytes, err := ioutil.ReadAll(result.Body)
	assert.Nil(err)
	bodyString := string(bodyBytes)

	assert.Equal("Please log in.\n", bodyString)
}

// -----------------------------------------------------------------------------
// Tests - Plugin triggers.
// -----------------------------------------------------------------------------

// TestWrongTrigger - Wrong trigger shouldn't do anything.
func TestWrongTrigger(t *testing.T) {
	resp, err := runTestPluginCommand(t, "/ihavenoideawhatimdoing")

	assert.Nil(t, resp)
	assert.Nil(t, err)
}

// TestRollHelp - Help command returns help text.
func TestRollHelp(t *testing.T) {
	resp, err := runTestPluginCommand(t, "/roll help")

	assert.NotNil(t, resp)
	assert.Nil(t, err)

	// TODO: Can these strings be moved out into something useful for
	// i18n?
	assert.True(t, strings.HasPrefix(resp.Text, `Support "any" [reasonable](https://en.wikipedia.org/wiki/Dice_notation) dice rolling request:`))
}

// TestRollEmpty - Empty /roll returns a snide comment.
func TestRollEmpty(t *testing.T) {
	resp, err := runTestPluginCommand(t, "/roll")

	assert.NotNil(t, resp)
	assert.Nil(t, err)

	// Negative tests.
	assert.False(t, strings.Contains(resp.Text, "rolls requested; I'm only doing 10."))

	// Positive tests.
	assert.True(t, strings.Contains(resp.Text, "throws the dice…"))
	assert.True(t, strings.Contains(resp.Text, "That accomplished nothing."))
}

// TestRollTooMany - /roll with too many requests includes a warning.
func TestRollTooMany(t *testing.T) {
	resp, err := runTestPluginCommand(t, "/roll 1 2 3 4 5 6 7 8 9 10 11")

	assert.NotNil(t, resp)
	assert.Nil(t, err)

	// Negative tests.
	assert.False(t, strings.Contains(resp.Text, "That accomplished nothing."))

	// Positive tests.
	assert.True(t, strings.Contains(resp.Text, "throws the dice…"))
	assert.True(t, strings.Contains(resp.Text, "rolls requested; I'm only doing 10."))
}

// TestSimpleRoll - /roll with a number.
func TestSimpleRoll(t *testing.T) {
	resp, err := runTestPluginCommand(t, "/roll 6")

	assert.NotNil(t, resp)
	assert.Nil(t, err)

	// Negative tests.
	assert.False(t, strings.Contains(resp.Text, "That accomplished nothing."))
	assert.False(t, strings.Contains(resp.Text, "rolls requested; I'm only doing 10."))

	// Positive tests.
	assert.True(t, strings.Contains(resp.Text, "throws the dice…"))
	assert.True(t, strings.Contains(resp.Attachments[0].Text, "1d6"))
}

// TestNormalRoll - /roll with an xdy value.
func TestNormalRoll(t *testing.T) {
	resp, err := runTestPluginCommand(t, "/roll 3d6")

	assert.NotNil(t, resp)
	assert.Nil(t, err)

	// Negative tests.
	assert.False(t, strings.Contains(resp.Text, "That accomplished nothing."))
	assert.False(t, strings.Contains(resp.Text, "rolls requested; I'm only doing 10."))

	// Positive tests.
	assert.True(t, strings.Contains(resp.Text, "throws the dice…"))
	assert.True(t, strings.Contains(resp.Attachments[0].Text, "3d6"))
}

// TestComboRoll - /roll with a combo.
func TestComboRoll(t *testing.T) {
	resp, err := runTestPluginCommand(t, "/roll dnd")

	assert.NotNil(t, resp)
	assert.Nil(t, err)

	// Negative tests.
	assert.False(t, strings.Contains(resp.Text, "That accomplished nothing."))
	assert.False(t, strings.Contains(resp.Text, "rolls requested; I'm only doing 10."))

	// Positive tests.
	assert.True(t, strings.Contains(resp.Text, "throws the dice…"))
	assert.True(t, strings.Contains(resp.Attachments[0].Text, "D&D standard:"))
}
