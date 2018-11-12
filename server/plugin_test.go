package main

import (
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
)

// -----------------------------------------------------------------------------
// Utilities
// -----------------------------------------------------------------------------

func runTestPluginCommand(t *testing.T, cmd string) (*model.CommandResponse, *model.AppError) {
	p := initTestPlugin(t)
	assert.Nil(t, p.OnActivate())

	var command *model.CommandArgs
	command = &model.CommandArgs{
		Command: cmd,
	}

	return p.ExecuteCommand(&plugin.Context{}, command)
}

func initTestPlugin(t *testing.T) *RollyPlugin {
	api := &plugintest.API{}
	api.On("RegisterCommand", mock.Anything).Return(nil)
	api.On("UnregisterCommand", mock.Anything, mock.Anything).Return(nil)
	api.On("GetUser", mock.Anything).Return(&model.User{
		Id:       "userid",
		Nickname: "User",
	}, (*model.AppError)(nil))

	p := RollyPlugin{}
	p.SetAPI(api)

	return &p
}

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

// TestPluginLifecycle -
func TestPluginLifecycle(t *testing.T) {
	p := initTestPlugin(t)

	assert.Nil(t, p.OnActivate())
	// TODO : Try ExecuteCommand while deactivated?
	assert.Nil(t, p.OnDeactivate())
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

// TestWrongTrigger - Wrong trigger shouldn't do anything.
func TestWrongTrigger(t *testing.T) {
	resp, err := runTestPluginCommand(t, "/ihavenoideawhatimdoing")

	assert.Nil(t, resp)
	assert.Nil(t, err)
}

// TestHelp - Help command returns help text.
func TestHelp(t *testing.T) {
	resp, err := runTestPluginCommand(t, "/roll help")

	assert.NotNil(t, resp)
	assert.Nil(t, err)

	// TODO: Can these strings be moved out into something useful for
	// i18n?
	assert.True(t, strings.HasPrefix(resp.Text, `Support "any" [reasonable](https://en.wikipedia.org/wiki/Dice_notation) dice rolling request:`))
}

// TestEmpty - Empty /roll returns a snide comment.
func TestEmpty(t *testing.T) {
	resp, err := runTestPluginCommand(t, "/roll")

	assert.NotNil(t, resp)
	assert.Nil(t, err)

	assert.True(t, strings.Contains(resp.Text, "throws the dice…"))
	assert.True(t, strings.Contains(resp.Text, "That accomplished nothing."))
}

// TestTooMany - /roll with too many requests includes a warning.
func TestTooMany(t *testing.T) {
	resp, err := runTestPluginCommand(t, "/roll 1 2 3 4 5 6 7 8 9 10 11")

	assert.NotNil(t, resp)
	assert.Nil(t, err)

	assert.True(t, strings.Contains(resp.Text, "throws the dice…"))
	assert.True(t, strings.Contains(resp.Text, "rolls requested; I'm only doing 10."))
}
