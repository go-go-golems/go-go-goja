package botcli

import (
	"fmt"
	"sort"
	"strings"
)

func ResolveBot(selector string, discovered []DiscoveredBot) (DiscoveredBot, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return DiscoveredBot{}, fmt.Errorf("bot selector is empty")
	}

	matches := make([]DiscoveredBot, 0, len(discovered))
	for _, bot := range discovered {
		fullPath := bot.FullPath()
		if fullPath == selector || bot.Verb.Name == selector || bot.Verb.FunctionName == selector || strings.HasSuffix(fullPath, " "+selector) {
			matches = append(matches, bot)
		}
	}

	switch len(matches) {
	case 0:
		return DiscoveredBot{}, fmt.Errorf("bot %q not found", selector)
	case 1:
		return matches[0], nil
	default:
		paths := make([]string, 0, len(matches))
		for _, bot := range matches {
			paths = append(paths, bot.FullPath())
		}
		sort.Strings(paths)
		return DiscoveredBot{}, fmt.Errorf("bot selector %q is ambiguous: %s", selector, strings.Join(paths, ", "))
	}
}

func ResolveBotFromArgs(args []string, discovered []DiscoveredBot) (DiscoveredBot, []string, error) {
	var lastErr error
	for prefixLen := len(args); prefixLen >= 1; prefixLen-- {
		selector := strings.Join(args[:prefixLen], " ")
		bot, err := ResolveBot(selector, discovered)
		if err == nil {
			return bot, args[prefixLen:], nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return DiscoveredBot{}, nil, lastErr
	}
	return DiscoveredBot{}, nil, fmt.Errorf("no bot selector provided")
}
