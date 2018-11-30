package main

import (
	"math/rand"
	"regexp"
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
		Id:        "userid",
		Nickname:  "User",
		Username:  "hunter2",
		FirstName: "User",
		LastName:  "McUserface",
	}, (*model.AppError)(nil))

	p := RollyPlugin{}
	p.SetAPI(api)

	return &p
}

// -----------------------------------------------------------------------------
// Tests - Plugin utilities.
// -----------------------------------------------------------------------------

// TestGetHelp - Help command returns help text.
func TestGetHelp(t *testing.T) {
	p := initTestPlugin(t)
	resp, err := p.GetHelp()

	assert.NotNil(t, resp)
	assert.Nil(t, err)

	// TODO: Can these strings be moved out into something useful for
	// i18n?
	assert.True(t, strings.HasPrefix(resp.Text, `Support "any" [reasonable](https://en.wikipedia.org/wiki/Dice_notation) dice rolling request:`))
}

// TestInit - Make sure Init() compiles the regexes.
func TestInit(t *testing.T) {
	p := initTestPlugin(t)

	assert.Nil(t, p.simplePattern)
	assert.Nil(t, p.comboPattern)
	assert.Nil(t, p.rollPattern)

	p.Init()

	assert.NotNil(t, p.simplePattern)
	assert.NotNil(t, p.comboPattern)
	assert.NotNil(t, p.rollPattern)
}

// TestGetCommand - How's this going to fail, really?
func TestGetCommand(t *testing.T) {
	p := initTestPlugin(t)

	cmd := p.GetCommand()

	assert.EqualValues(t, cmd.Trigger, "roll")
	assert.EqualValues(t, cmd.Description, "Roll one or more dice. With combos!")
}

// TestSeedRng - Seed the default RNG.
func TestSeedRng(t *testing.T) {
	p := initTestPlugin(t)
	p.SeedRng()

	assert.NotNil(t, p) // Not actually testing anything useful.
}

// TestGetRandom - Make sure GetRandom() is returning values.
func TestGetRandom(t *testing.T) {
	p := initTestPlugin(t)
	rand.Seed(0) // Make these deterministic.

	val := p.GetRandom(1)
	assert.EqualValues(t, val, 1)

	val = p.GetRandom(6)
	assert.True(t, val >= 1 && val <= 6)
	assert.EqualValues(t, val, 1)
}

// TestGetName - Make sure GetName() does something suitable.
func TestGetName(t *testing.T) {
	p := initTestPlugin(t)

	// TODO: How to mock this better so we can test users with missing fields?
	name, err := p.GetName("userid")
	assert.Nil(t, err)
	assert.EqualValues(t, name, "User")
}

// -----------------------------------------------------------------------------
// Tests - Other functions.
// -----------------------------------------------------------------------------

// TestSum - Make sure sum() sums the contents.
func TestSum(t *testing.T) {
	assert.EqualValues(t, sum([]int{}), 0)
	assert.EqualValues(t, sum([]int{1}), 1)
	assert.EqualValues(t, sum([]int{1, -1}), 0)
	assert.EqualValues(t, sum([]int{1, 2, 3}), 6)
}

// TestMin - Make sure min returns the minimum.
func TestMin(t *testing.T) {
	assert.EqualValues(t, min(1, 2), 1)
	assert.EqualValues(t, min(2, 1), 1)
}

// TestFindNamedSubstrings - Make sure regexes can be turned into dicts.
//
// This assumes you've already checked to see if there's a match inside.
func TestFindNamedSubstrings(t *testing.T) {
	// Test with the simpleRegex.
	pattern := regexp.MustCompile(simpleRegex)
	matches := FindNamedSubstrings(pattern, "6")
	assert.EqualValues(t, matches["num_sides"], "6")

	// Test with the comboRegex.
	pattern = regexp.MustCompile(comboRegex)
	matches = FindNamedSubstrings(pattern, "dnd")
	assert.EqualValues(t, matches["combo_name"], "dnd")

	// Test with the rollRegex.
	pattern = regexp.MustCompile(rollRegex)
	matches = FindNamedSubstrings(pattern, "1d2+3")
	assert.EqualValues(t, matches["num_dice"], "1")
	assert.EqualValues(t, matches["num_sides"], "2")
	assert.EqualValues(t, matches["modifier"], "+")
	assert.EqualValues(t, matches["modifier_value"], "3")
}
