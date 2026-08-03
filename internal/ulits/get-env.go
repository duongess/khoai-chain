package utils

import (
	"embed"
	"fmt"
	"sync"

	"github.com/joho/godotenv"
)

var (
	sourceCode embed.FS
	envCache   map[string]string
	once       sync.Once
	envErr     error
)

func SetSourceCode(source embed.FS) {
	sourceCode = source
}

func GetEnv(key string) (string, error) {
	once.Do(func() {
		var envData []byte
		envData, envErr = sourceCode.ReadFile(".env")
		if envErr == nil {
			envCache, envErr = godotenv.Unmarshal(string(envData))
		}
	})

	if envErr != nil {
		return "", envErr
	}

	val, exists := envCache[key]
	if !exists {
		return "", fmt.Errorf("key %s does not exist in .env", key)
	}

	return val, nil
}
