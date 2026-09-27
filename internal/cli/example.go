package cli

import "dotenvless/internal/application"

func writeExample(root string, keys []string) error { return application.WriteExample(root, keys) }
