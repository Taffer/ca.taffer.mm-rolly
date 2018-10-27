# ca.taffer.mm-rolly

Dice rolling plugin for Mattermost.

Inspired by
[moussetc/mattermost-plugin-dice-roller](https://github.com/moussetc/mattermost-plugin-dice-roller)
and [DiceBot](https://dice-b.appspot.com/) for Slack.

Note that I don't actually know Go, so this could be rough...

## Goals

Support "any" reasonable dice rolling request:

* *x*d*y* or *x*D*y* to roll a *y* sided die *x* times
* modifiers: *x*d*y*+*z*, *x*d*y*-*z* (with a minimum of 1)
* *x*d% - same as *x*d100
* *x*d*y*/*z* - divide the result by *z*
* *x*d*y*<1 - discards the lowest roll (so 4d6<1 would return a value between 3 and 18)

Nerd combos:

* dnd - same as 3d6 six times
* dnd+ - same as 4d6<1 six times

Number of dice per roll will be limited to 100 so malicious users can't flood
the channel with dice output.

## Credits

* [Mattermost's plugin sample](https://github.com/mattermost/mattermost-plugin-sample)
