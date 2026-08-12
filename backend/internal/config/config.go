package config

import (
	"fmt"
	"os"
	"strings"
)

// RequiredSecret reads a required value either directly from NAME or from the
// file referenced by NAME_FILE. The two forms are mutually exclusive so a
// stale environment value cannot silently override a mounted secret.
func RequiredSecret(name string) (string, error) {
	direct := strings.TrimSpace(os.Getenv(name))
	file := strings.TrimSpace(os.Getenv(name + "_FILE"))
	if direct != "" && file != "" {
		return "", fmt.Errorf("configure apenas %s ou %s_FILE", name, name)
	}
	if file != "" {
		raw, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("ler %s_FILE: %w", name, err)
		}
		if value := strings.TrimSpace(string(raw)); value != "" {
			return value, nil
		}
		return "", fmt.Errorf("%s_FILE aponta para um arquivo vazio", name)
	}
	if direct == "" {
		return "", fmt.Errorf("%s ou %s_FILE é obrigatório", name, name)
	}
	return direct, nil
}
