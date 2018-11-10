package main

import (
	"fmt"
	"math/rand"
	"net/http"
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
}

// -----------------------------------------------------------------------------
// Constants
// -----------------------------------------------------------------------------

const (
	pluginPath string = "plugins/ca.taffer.mm-rolly" // Maybe?
	iconFile   string = "if_die_1055072.png"
	iconPath   string = pluginPath + "/" + iconFile
	iconURI    string = "/" + iconPath

	trigger string = "rolly"
)

// -----------------------------------------------------------------------------
// Mattermost events.
// -----------------------------------------------------------------------------

// OnActivate - Register the plugin.
func (p *RollyPlugin) OnActivate() error {
	p.active = true

	// Handle requests for the icon file.
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

	text := fmt.Sprintf("Command was: %s…", args.Command)

	attachments := []*model.SlackAttachment{
		{
			Text:     "Attachment: " + text,
			Fallback: fmt.Sprintf("Fallback: %s rolled…", user.GetFullName()),
			ThumbURL: iconURI,
		},
	}

	props := map[string]interface{}{
		"from_webhook":  "true",
		"use_user_icon": "true",
	}

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		Attachments:  attachments,
		Username:     user.Username,
		Text:         "Response: " + text,
		Props:        props,
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
		DisplayName:      "Rolly",
		AutoComplete:     true,
		AutoCompleteDesc: "🎲 Roll the dice! Use `/" + trigger + " help` for usage.",
		AutoCompleteHint: "6 d10 2d4+2 and other modifiers.",
		IconURL:          iconURI,
	}
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

If *x* isn't specified, it defaults to 1.`

	props := map[string]interface{}{
		"from_webhook":  "true",
		"use_user_icon": "true",
	}

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         helpText,
		Props:        props,
		Username:     "Rolly",
		IconURL:      iconURI,
	}, nil
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

// GetRandomIntn - Gets a random number from [0, n).
func (p *RollyPlugin) GetRandomIntn(n int) int {
	return rand.Intn(n)
}
