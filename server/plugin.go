package main

import (
	"math/rand"
	"regexp"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// RollyPlugin instance.
type RollyPlugin struct {
	plugin.MattermostPlugin

	// URL router/dispatcher.
	router *mux.Router

	// Is this active?
	active bool

	// Dice rolling patterns.
	simplePattern *regexp.Regexp
	comboPattern  *regexp.Regexp
	rollPattern   *regexp.Regexp
}

// -----------------------------------------------------------------------------
// Constants
// -----------------------------------------------------------------------------

const (
	pluginPath string = "plugins/ca.taffer.mm-rolly" // Maybe?
	iconFile   string = "if_die_1055072.png"
	iconPath   string = pluginPath + "/" + iconFile
	iconURI    string = "/" + iconPath
	repoURI    string = "https://github.com/Taffer/ca.taffer.mm-rolly"

	trigger    string = "roll"
	pluginName string = "Rolly"

	simpleRegex string = `^(?P<num_sides>[0-9\%F]+)$`
	comboRegex  string = `(?i)^((?P<combo_name>(d[n&]d\+?|open)))$`
	rollRegex   string = `(?i)^((?P<num_dice>[0-9]+)?d)?(?P<num_sides>[0-9\%F]+)((?P<modifier>[+-/<>x*!])(?P<modifier_value>[0-9]*))?$`
)

// -----------------------------------------------------------------------------
// Different commands the roller knows.
// -----------------------------------------------------------------------------

// GetHelp - Produce a help message when the user asks for help.
func (p *RollyPlugin) GetHelp() (*model.CommandResponse, *model.AppError) {
	helpText := `Support "any" [reasonable](https://en.wikipedia.org/wiki/Dice_notation) dice rolling request:

* *x*d*y* or *x*D*y* to roll a *y* sided die *x* times
* modifiers: *x*d*y*+*z* (supported modifiers: +, -, x or *, /)
* exploding dice (for every max value, roll and add): *x*d*y*!
* *x*d% - same as *x*d100
* *x*dF - roll
  [FUDGE](https://en.wikipedia.org/wiki/Fudge_%28role-playing_game_system%29)
  dice
* *x*d*y*<*z* - discards the lowest *z* rolls (so 4d6<1 would return a value
  between 3 and 18)
* *x*d*y*>*z* - keeps the best *z* rolls (so 4d6>1 would return a value
  between 1 and 6)

If *x* isn't specified, it defaults to 1. If *y* is less than 2, it defaults
to 2. If you specify a modifier, you must also specify a *z* value.

Supports these nerd combos:

* dnd - same as 3d6 six times (standard D&D or Pathfinder)
* dnd+ - same as 4d6<1 six times (common house rule for D&D or Pathfinder)
* open - roll d%, if it's >= 95, roll again and add, repeating if necessary`

	props := map[string]interface{}{
		"from_webhook":  "true",
		"use_user_icon": "true",
	}

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         helpText,
		Props:        props,
		Username:     pluginName,
		IconURL:      iconURI,
	}, nil
}

// -----------------------------------------------------------------------------
// Utility functions.
// -----------------------------------------------------------------------------

// Init - Initialize plugin.
func (p *RollyPlugin) Init() {
	// Dice rolling patterns.
	//
	// See prototype.py for more readable (?) versions of these.
	p.simplePattern = regexp.MustCompile(simpleRegex)
	p.comboPattern = regexp.MustCompile(comboRegex)
	p.rollPattern = regexp.MustCompile(rollRegex)
}

// GetCommand - Return the Command to register.
func (p *RollyPlugin) GetCommand() *model.Command {
	return &model.Command{
		Trigger:          trigger,
		Description:      "Roll one or more dice. With combos!",
		DisplayName:      pluginName,
		AutoComplete:     true,
		AutoCompleteDesc: "🎲 Roll the dice! Use `/" + trigger + " help` for usage.",
		AutoCompleteHint: "6 d10 2d4+2 and other modifiers.",
		IconURL:          iconURI,
	}
}

// GetName - What should we call the user?
func (p *RollyPlugin) GetName(userID string) (string, *model.AppError) {
	user, userErr := p.API.GetUser(userID)
	if userErr != nil {
		return "Error McErrorface", userErr
	}

	name := user.Nickname
	if len(name) < 1 {
		name = user.Username

		if len(name) < 1 {
			name = user.FirstName + " " + user.LastName
		}
	}

	return name, nil
}

// Is there already a way to do this?
func sum(values []int) int {
	total := 0
	for idx := 0; idx < len(values); idx++ {
		total += values[idx]
	}

	return total
}

// Is there already a way to do this?
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// FindNamedSubstrings - Return a map of named matches.
func FindNamedSubstrings(re *regexp.Regexp, candidate string) map[string]string {
	found := make(map[string]string)

	values := re.FindStringSubmatch(candidate)
	keys := re.SubexpNames()

	// Why do you start indexing keys at 1 instead of 0?
	for idx := 1; idx < len(keys); idx++ {
		found[keys[idx]] = values[idx]
	}

	return found
}

// -----------------------------------------------------------------------------
// Random number generator.
//
// Created as methods rather than just calling rand.Seed(), etc. directly
// because I'd like to make a fork of this that uses the ISARA Radiate toolkit
// RNGs to generate cryptographically secure random numbers. Massive overkill
// for this sort of application!
// -----------------------------------------------------------------------------

// SeedRng - Seed the random number generator.
func (p *RollyPlugin) SeedRng() {
	rand.Seed(time.Now().UnixNano())
}

// GetRandom - Gets a random number from [1, n].
func (p *RollyPlugin) GetRandom(n int) int {
	return rand.Intn(n) + 1
}
