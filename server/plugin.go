package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// URL router/dispatcher.
	router *mux.Router
}

// -----------------------------------------------------------------------------
// Constants
// -----------------------------------------------------------------------------
const (
	trigger    string = "rolly"
	pluginPath string = "plugins/ca.taffer.mm-rolly.mattermost.plugin.rolly" // Maybe?
	iconFile   string = "if_die_1055072.png"
	iconPath   string = pluginPath + "/" + iconFile
	iconUri    string = "/" + iconPath
)

// Register the plugin when it gets activated.
func (p *Plugin) OnActivate() error {
	// Handle requests for the icon file.
	p.router = mux.NewRouter()
	p.router.HandleFunc("/"+iconFile, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, iconPath)
	})

	// Register our command handler.
	return p.API.RegisterCommand(&model.Command{
		Trigger:          trigger,
		Description:      "Roll one or more dice. With combos!",
		DisplayName:      "Rolly",
		AutoComplete:     true,
		AutoCompleteDesc: "Roll the dice! 🎲 Use `/" + trigger + " help` for usage.",
		AutoCompleteHint: "6 d10 2d4+2 and other modifiers.",
		IconURL:          iconUri,
	})
}

// Handle HTTP requests.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Mattermost-User-Id") == "" {
		http.Error(w, "Please log in.", http.StatusForbidden)
		return
	}

	p.router.ServeHTTP(w, r)
}

// Handle commands.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if strings.HasPrefix(args.Command, "/"+trigger) {
		text := "d666!!!! 🤘"

		// Get the user to we can display the right name
		user, userErr := p.API.GetUser(args.UserId)
		if userErr != nil {
			return nil, userErr
		}

		attachments := []*model.SlackAttachment{
			{
				Text:     text,
				Fallback: fmt.Sprintf("%s rolled…", user.GetFullName()),
				ThumbURL: iconUri,
			},
		}

		props := map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
		}

		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
			Attachments:  attachments,
			Props:        props,
		}, nil
	}

	return nil, nil
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
