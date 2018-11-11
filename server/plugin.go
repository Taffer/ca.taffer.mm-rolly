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

	trigger    string = "rolly" // TODO: Change to "roll" for release.
	pluginName string = "Rolly"
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

	// Dice rolling patterns.
	//
	// See prototype.py for more readable (?) versions of these.
	p.simplePattern = regexp.MustCompile(`^(?P<num_sides>[0-9\%]+)$`)
	p.comboPattern = regexp.MustCompile(`(?i)^((?P<combo_name>(d[n&]d|open))(?P<combo_flag>\+)?)$`)
	p.rollPattern = regexp.MustCompile(`(?i)^((?P<num_dice>[0-9]+)?d)?(?P<num_sides>[0-9\%]+)((?P<modifier>[+-/<])(?P<modifier_value>[0-9]+))?$`)

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

	// Get the user to we can display the right name
	user, userErr := p.API.GetUser(args.UserId)
	if userErr != nil {
		return nil, userErr
	}

	responseText := fmt.Sprintf("%s throws the dice…", user.Nickname)

	rolls := strings.Fields(args.Command)[1:]
	if len(rolls) > 10 {
		rolls = rolls[0:11]
		responseText += fmt.Sprintf("\n⚠️ %d rolls requested; I'm only doing 10.", len(rolls))
	}

	attachments := []*model.SlackAttachment{}
	if len(rolls) == 0 {
		responseText += fmt.Sprintf("\n🚫 That accomplished nothing.")
	} else {
		rollText := "Results:"
		for idx := 0; idx < len(rolls); idx++ {
			if p.simplePattern.MatchString(rolls[idx]) == true {
				matches := FindNamedSubstrings(p.simplePattern, rolls[idx])

				_, total := p.RollDice(1, matches["num_sides"], "", 0)

				rollText += fmt.Sprintf("\n🎲 \"1d%v\" **%d**", rolls[idx], total)
			} else if p.comboPattern.MatchString(rolls[idx]) == true {
				rollText += fmt.Sprintf("\n🎲 Combo **%v** isn't implemented yet, sorry.", rolls[idx])
			} else if p.rollPattern.MatchString(rolls[idx]) == true {
				matches := FindNamedSubstrings(p.rollPattern, rolls[idx])

				numDice, err := strconv.Atoi(matches["num_dice"])
				if err != nil {
					// This is optional, so it might be empty.
					numDice = 1
				}
				if numDice > 100 {
					numDice = 100
				}
				sides := matches["num_sides"] // Left as string for d% rolls.
				modifier := matches["modifier"]
				modifierValue, err := strconv.Atoi(matches["modifier_value"])

				dice, total := p.RollDice(numDice, sides, modifier, modifierValue)

				rollText += fmt.Sprintf("\n🎲 %q = **%d** %v", rolls[idx], total, dice)
			} else {
				rollText += fmt.Sprintf("\n🎲 I have no idea what to do with %q.", rolls[idx])
			}
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
* modifiers: *x*d*y*+*z*, *x*d*y*-*z* (with a minimum of 1)
* *x*d% - same as *x*d100
* *x*d*y*/*z* - divide the result by *z*
* *x*d*y*<1 - discards the lowest roll (so 4d6<1 would return a value between 3 and 18)

If *x* isn't specified, it defaults to 1. Also supports nerd combos:

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

// Is there already a way to do this?
func sum(values []int) int {
	total := 0
	for idx := 0; idx < len(values); idx++ {
		total += values[idx]
	}

	return total
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
	} else {
		dieSides, _ = strconv.Atoi(sides)
		if dieSides < 2 {
			dieSides = 2
		}
	}

	for idx := 0; idx < dice; idx++ {
		rolls = append(rolls, p.GetRandom(dieSides))
	}

	sort.Ints(rolls)
	total := sum(rolls)

	// Most of the supported modifiers are trivial.
	switch modifier {
	case "+":
		total += modifierValue
	case "-":
		total -= modifierValue
	case "/":
		total /= modifierValue
	case "<":
		// Currently only the lowest is thrown out.
		if len(rolls) > 1 {
			total = sum(rolls[1:])
		}
	}

	// Clamp rolls to a minimum of 1.
	if total < 1 {
		total = 1
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
