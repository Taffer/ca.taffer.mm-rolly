package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
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
				Text:       rollText,
				Color:      "#76C2AF",
				Fallback:   "🎲",
				ThumbURL:   iconURI,
				AuthorName: pluginName,
				AuthorIcon: iconURI,
				AuthorLink: repoURI,
			},
		}
	}

	props := map[string]interface{}{
		"from_webhook":  "true",
		"use_user_icon": "false", // I thought setting this to false would make it use iconURI, but no?
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
