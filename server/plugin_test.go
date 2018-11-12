package main

import (
	"io/ioutil"
	"math/rand"
	"net/http/httptest"
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

// TestHandleRoll - Make sure different sorts of rolls do things.
func TestHandleRoll(t *testing.T) {
	p := initTestPlugin(t)
	p.Init()

	// simplePattern matches
	rand.Seed(0) // Make these deterministic.
	response := p.HandleRoll("6", "")
	assert.EqualValues(t, response, `"1d6" = **1**`)

	response = p.HandleRoll("%", "")
	assert.EqualValues(t, response, `"1d%" = **15**`)

	response = p.HandleRoll("F", "")
	assert.EqualValues(t, response, `"1dF" = **0**`)

	// comboPattern matches
	rand.Seed(0) // Make these deterministic.
	response = p.HandleRoll("dnd", "")
	assert.EqualValues(t, response, "D&D standard:\n* 3d6 [1 1 2] = **4**\n* 3d6 [5 5 6] = **16**\n* 3d6 [1 2 6] = **9**\n* 3d6 [1 1 6] = **8**\n* 3d6 [1 1 6] = **8**\n* 3d6 [1 3 6] = **10**")

	response = p.HandleRoll("dnd+", "")
	assert.EqualValues(t, response, "D&D variant:\n* 4d6<1 [1 1 5 6] = **12**\n* 4d6<1 [1 1 3 6] = **10**\n* 4d6<1 [1 2 5 5] = **12**\n* 4d6<1 [3 4 5 6] = **15**\n* 4d6<1 [2 3 3 5] = **11**\n* 4d6<1 [1 1 2 6] = **9**")

	response = p.HandleRoll("open", "")
	assert.EqualValues(t, response, "Rolemaster open-ended: 1d% [77] = **77**")

	// rollPattern matches
	rand.Seed(0) // Make these deterministic.
	response = p.HandleRoll("d3", "")
	assert.EqualValues(t, response, `"d3" = **1**`)

	response = p.HandleRoll("2d3", "")
	assert.EqualValues(t, response, `"2d3" [1 2] = **3**`)

	response = p.HandleRoll("1000d6", "")
	assert.EqualValues(t, response, "1000 is too many, rolling 100.\n\"1000d6\" [1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 2 2 2 2 2 2 2 2 2 2 2 2 2 3 3 3 3 3 3 3 3 3 3 3 3 3 3 3 3 3 3 3 4 4 4 4 4 4 4 4 4 4 5 5 5 5 5 5 5 5 5 5 5 5 5 5 5 5 6 6 6 6 6 6 6 6 6 6 6 6 6 6 6] = **320**")

	response = p.HandleRoll("6!", "")
	assert.EqualValues(t, response, "\"6!\" [6 1] = **7**")

	response = p.HandleRoll("monkey", "")
	assert.EqualValues(t, response, "I have no idea what to do with this: monkey")
}

// TestRollDice - Make sure different combinations return correct values.
func TestRollDice(t *testing.T) {
	p := initTestPlugin(t)
	p.Init()

	// Different number of sides.
	rand.Seed(0) // Make these deterministic.
	rolls, total := p.RollDice(1, "%", "", 0)
	assert.EqualValues(t, rolls[0], 75)
	assert.EqualValues(t, total, 75)

	rolls, total = p.RollDice(1, "F", "", 0)
	assert.EqualValues(t, rolls[0], -1)
	assert.EqualValues(t, total, -1)

	rolls, total = p.RollDice(1, "1", "", 0)
	assert.EqualValues(t, rolls[0], 2)
	assert.EqualValues(t, total, 2)

	// Different modifiers.
	rand.Seed(0) // Make these deterministic.
	rolls, total = p.RollDice(1, "6", "+", 1)
	assert.EqualValues(t, rolls[0], 1)
	assert.EqualValues(t, total, 2)

	rolls, total = p.RollDice(1, "6", "-", 6)
	assert.EqualValues(t, rolls[0], 1)
	assert.EqualValues(t, total, 1)

	rolls, total = p.RollDice(1, "6", "/", 2)
	assert.EqualValues(t, rolls[0], 2)
	assert.EqualValues(t, total, 1)

	rolls, total = p.RollDice(1, "6", "x", 2)
	assert.EqualValues(t, rolls[0], 5)
	assert.EqualValues(t, total, 10)

	rand.Seed(0) // Make these deterministic.
	rolls, total = p.RollDice(2, "6", "<", 1)
	assert.EqualValues(t, rolls[0], 1)
	assert.EqualValues(t, rolls[1], 1)
	assert.EqualValues(t, total, 1)

	rolls, total = p.RollDice(2, "6", ">", 1)
	assert.EqualValues(t, rolls[0], 2)
	assert.EqualValues(t, rolls[1], 5)
	assert.EqualValues(t, total, 5)

	rolls, total = p.RollDice(2, "6", ">", 3)
	assert.EqualValues(t, rolls[0], 5)
	assert.EqualValues(t, rolls[1], 6)
	assert.EqualValues(t, total, 11)

	rolls, total = p.RollDice(2, "6", ">", 0)
	assert.EqualValues(t, rolls[0], 2)
	assert.EqualValues(t, rolls[1], 6)
	assert.EqualValues(t, total, 0)

	// Need to roll these several times before it actually explodes.
	rand.Seed(0) // Make these deterministic.
	rolls, total = p.RollDice(1, "F", "!", 0)
	rolls, total = p.RollDice(1, "F", "!", 0)
	rolls, total = p.RollDice(1, "F", "!", 0)
	rolls, total = p.RollDice(1, "F", "!", 0)
	rolls, total = p.RollDice(1, "F", "!", 0)
	assert.EqualValues(t, rolls[0], 1)
	assert.EqualValues(t, rolls[1], 0)
	assert.EqualValues(t, total, 1)
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
