package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
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

	trigger    string = "roll"
	pluginName string = "Rolly"

	simpleRegex string = `^(?P<num_sides>[0-9\%F]+)$`
	comboRegex  string = `(?i)^((?P<combo_name>(d[n&]d\+?|open)))$`
	rollRegex   string = `(?i)^((?P<num_dice>[0-9]+)?d)?(?P<num_sides>[0-9\%F]+)((?P<modifier>[+-/<>x*!])(?P<modifier_value>[0-9]*))?$`
)

// -----------------------------------------------------------------------------
// Mattermost events.
// -----------------------------------------------------------------------------

// OnActivate - Register the plugin.
//
// Is this called once when the plugin is loaded? Repeatedly? Whenever you
// switch from "Deactivated" to "Activated"?
func (p *RollyPlugin) OnActivate() error {
	p.active = true

	p.Init()

	// Handle requests for the icon file.  Maybe this nil check isn't necessary
	// and OnActivate() is only called once per lifetime?
	if p.router == nil {
		p.router = mux.NewRouter()
		p.router.HandleFunc("/"+iconFile, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, iconPath)
		})
	}

	// Register our command handler.
	err := p.API.RegisterCommand(p.GetCommand())

	return err
}

// OnDeactivate - No more work...
func (p *RollyPlugin) OnDeactivate() error {
	p.active = false

	return nil
}

// ServeHTTP - Handle HTTP requests.
func (p *RollyPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Mattermost-User-Id") == "" {
		http.Error(w, "Please log in.", http.StatusForbidden)
		return
	}

	p.router.ServeHTTP(w, r)
}

// ExecuteCommand - Handle commands.
func (p *RollyPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if strings.HasPrefix(args.Command, "/"+trigger) == false {
		// It's not for us.
		return nil, nil
	}

	if strings.Contains(args.Command, "help") {
		return p.GetHelp()
	}

	// Get the user to we can display the right name.
	userName, userErr := p.GetName(args.UserId)
	if userErr != nil {
		return nil, userErr
	}

	responseText := fmt.Sprintf("%s throws the dice…", userName)

	rolls := strings.Fields(args.Command)[1:]
	if len(rolls) > 10 {
		rolls = rolls[0:11]
		responseText += fmt.Sprintf("\n⚠️ %d rolls requested; I'm only doing 10.", len(rolls))
	}

	attachments := []*model.SlackAttachment{}
	if len(rolls) == 0 {
		responseText += fmt.Sprintf("\n🚫 That accomplished nothing.")
	} else {
		rollText := ""
		for idx := 0; idx < len(rolls); idx++ {
			rollText += "\n🎲 "
			rollText = p.HandleRoll(rolls[idx], rollText)
		}

		attachments = []*model.SlackAttachment{
			{
				Text:     rollText,
				Fallback: "🎲",
				ThumbURL: iconURI,
			},
		}
	}

	props := map[string]interface{}{
		"from_webhook":  "true",
		"use_user_icon": "true",
	}

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		Attachments:  attachments,
		Username:     pluginName,
		Text:         responseText,
		Props:        props,
		IconURL:      iconURI,
	}, nil
}

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

// HandleRoll - Handle a rolling command.
//
// Returns the adjusted roll output.
func (p *RollyPlugin) HandleRoll(rollArg string, rollText string) string {
	if p.simplePattern.MatchString(rollArg) == true {
		// Simple roll (number only).
		matches := FindNamedSubstrings(p.simplePattern, rollArg)

		if matches["num_sides"] == "1" {
			rollText += "Your one-sided die rolls off into the shadows."
		} else {
			dice, total := p.RollDice(1, matches["num_sides"], "", 0)

			if len(dice) == 1 {
				rollText += fmt.Sprintf("\"1d%v\" = **%d**", rollArg, total)
			} else {
				rollText += fmt.Sprintf("%q %v = **%d**", rollArg, dice, total)
			}
		}

	} else if p.comboPattern.MatchString(rollArg) == true {
		// C-C-C-C-COMBO roll.
		matches := FindNamedSubstrings(p.comboPattern, rollArg)

		comboName := strings.ToLower(matches["combo_name"])
		switch comboName {
		case "dnd", "d&d":
			// D&D/Pathfinder: 3d6 for each stat.
			rollText += "D&D standard:"

			for idx := 0; idx < 6; idx++ {
				dice, total := p.RollDice(3, "6", "", 0)
				rollText += fmt.Sprintf("\n* 3d6 %v = **%d**", dice, total)
			}
		case "dnd+", "d&d+":
			// Common D&D/Pathfinder house rule: 4d6<1 for each stat.
			rollText += "D&D variant:"

			for idx := 0; idx < 6; idx++ {
				dice, total := p.RollDice(4, "6", "<", 1)
				rollText += fmt.Sprintf("\n* 4d6<1 %v = **%d**", dice, total)
			}
		case "open":
			// Rolemaster open-ended d%.
			dice, total := p.RollDice(1, "%", "", 0)
			allDice := dice
			for total >= 95 {
				dice, total = p.RollDice(1, "%", "", 0)
				allDice = append(allDice, dice[0])
			}

			sort.Ints(allDice)
			total = sum(allDice)
			rollText += fmt.Sprintf("Rolemaster open-ended: 1d%% %v = **%d**", allDice, total)
		default:
			// You can't actually reach this with the current regex.
			rollText += fmt.Sprintf("Combo **%v** isn't implemented yet, sorry.", rollArg)
		}

	} else if p.rollPattern.MatchString(rollArg) == true {
		// Typical roll (number of dice, sides, optional modifiers).
		matches := FindNamedSubstrings(p.rollPattern, rollArg)

		numDice, err := strconv.Atoi(matches["num_dice"])
		if err != nil {
			// This is optional, so it might be empty.
			numDice = 1
		}
		if numDice > 100 {
			rollText += fmt.Sprintf("%v is too many, rolling 100.\n", numDice)
			numDice = 100
		}
		if numDice < 1 {
			rollText += fmt.Sprintf("%v is too few, rolling 1.\n", numDice)
			numDice = 1
		}
		sides := matches["num_sides"] // Left as string for d% rolls.
		if sides == "1" {
			rollText += "Your one-sided die rolls off into the shadows."
		} else {
			modifier := matches["modifier"]
			modifierValue, err := strconv.Atoi(matches["modifier_value"])
			if err != nil {
				modifierValue = 0 // One wasn't specified. Blame the ! modifier.
			}

			dice, total := p.RollDice(numDice, sides, modifier, modifierValue)

			if len(dice) == 1 {
				rollText += fmt.Sprintf("%q = **%d**", rollArg, total)
			} else {
				rollText += fmt.Sprintf("%q %v = **%d**", rollArg, dice, total)
			}
		}
	} else {
		rollText += fmt.Sprintf("I have no idea what to do with this: %v", rollArg)
	}

	return rollText
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

// RollDice - Roll {dice}d{sides}{modifier}{modifier_value}.
//
// Returns an array of rolls, and the (modified) total.
func (p *RollyPlugin) RollDice(dice int, sides string, modifier string, modifierValue int) ([]int, int) {
	var rolls []int

	// Valid dieSides are digits, or %.
	var dieSides int
	if sides == "%" {
		dieSides = 100
	} else if sides == "F" {
		dieSides = 3
	} else {
		dieSides, _ = strconv.Atoi(sides)
		if dieSides < 2 {
			dieSides = 2
		}
	}

	for idx := 0; idx < dice; idx++ {
		value := p.GetRandom(dieSides)
		if sides == "F" {
			value -= 2 // FUDGE dice product -1, 0, 1
		}
		rolls = append(rolls, value)
	}

	sort.Ints(rolls)
	total := sum(rolls)

	// Most of the supported modifiers are trivial.
	switch modifier {
	case "+":
		total += modifierValue
	case "-":
		total -= modifierValue
		if total < 1 && sides != "F" {
			total = 1 // Clamp to 1, unless FUDGE.
		}
	case "/":
		if modifierValue > 0 {
			total /= modifierValue
		}
	case "x", "*":
		total *= modifierValue
	case "<": // Ignore the lowest modifierValue rolls.
		cutoff := min(modifierValue, len(rolls)-1)

		total = sum(rolls[cutoff:])
	case ">": // Keep the best modifierValue rolls.
		if modifierValue >= len(rolls) {
			total = sum(rolls)
		} else if modifierValue < 1 {
			total = 0
		} else {
			cutoff := len(rolls) - modifierValue

			total = sum(rolls[cutoff:])
		}
	case "!": // Exploding dice!
		explode := 0
		for idx := len(rolls) - 1; idx >= 0; idx-- {
			if (sides == "F" && rolls[idx] == 1) || rolls[idx] == dieSides {
				explode++
			}
		}

		for idx := 0; idx < explode; idx++ {
			boom := p.GetRandom(dieSides)
			if sides == "F" { // FUDGE is a special case.
				boom -= 2
				if boom == 1 {
					explode++
				}
			} else { // Normal case.
				if boom == dieSides {
					explode++
				}
			}
			rolls = append(rolls, boom)
		}

		total = sum(rolls)
	}

	return rolls, total
}

// SeedRng - Seed the random number generator.
func (p *RollyPlugin) SeedRng() {
	rand.Seed(time.Now().UnixNano())
}

// GetRandom - Gets a random number from [1, n].
func (p *RollyPlugin) GetRandom(n int) int {
	return rand.Intn(n) + 1
}
