package cli

import "strings"

func importArguments(args []string) (path string, overwrite, valid bool) {
	if len(args) < 2 || len(args) > 3 || args[0] != "import" {
		return "", false, false
	}
	for _, arg := range args[1:] {
		if arg == "--overwrite" {
			if overwrite {
				return "", false, false
			}
			overwrite = true
		} else {
			if path != "" || strings.HasPrefix(arg, "--") {
				return "", false, false
			}
			path = arg
		}
	}
	return path, overwrite, path != ""
}
