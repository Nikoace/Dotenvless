package runner

import (
	"sort"
	"strings"
)

func environment(base []string, secrets map[string]string) []string {
	entries := map[string]string{}
	for _, entry := range base {
		offset := 0
		if strings.HasPrefix(entry, "=") {
			offset = 1
		}
		index := strings.IndexByte(entry[offset:], '=')
		if index < 0 {
			continue
		}
		index += offset
		entries[strings.ToUpper(entry[:index])] = entry
	}
	keys := make([]string, 0, len(secrets))
	for key := range secrets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		normalized := strings.ToUpper(key)
		entries[normalized] = normalized + "=" + secrets[key]
	}
	keys = keys[:0]
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		env = append(env, entries[key])
	}
	return env
}
