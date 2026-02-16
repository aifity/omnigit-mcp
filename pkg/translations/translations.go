package translations

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/github/github-mcp-server/pkg/bodyfilter"
	"github.com/spf13/viper"
)

type TranslationHelperFunc func(key string, defaultValue string) string

func NullTranslationHelper(_ string, defaultValue string) string {
	return defaultValue
}

func TranslationHelper() (TranslationHelperFunc, func()) {
	var translationKeyMap = map[string]string{}
	v := viper.New()

	// Load from JSON file
	v.SetConfigName("github-mcp-server-config")
	v.SetConfigType("json")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		// ignore error if file not found as it is not required
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("Could not read JSON config: %v", err)
		}
	}

	// Load filter patterns from config if present
	if v.IsSet("filter_patterns") {
		patterns := v.GetStringSlice("filter_patterns")
		if len(patterns) > 0 {
			log.Printf("Loading %d filter patterns from config", len(patterns))
			bodyfilter.SetFilterPatterns(patterns)
		}
	}

	// create a function that takes both a key, and a default value and returns either the default value or an override value
	return func(key string, defaultValue string) string {
			key = strings.ToUpper(key)
			if value, exists := translationKeyMap[key]; exists {
				return value
			}
			// check if the env var exists
			if value, exists := os.LookupEnv("GITHUB_MCP_" + key); exists {
				// TODO I could not get Viper to play ball reading the env var
				translationKeyMap[key] = value
				return value
			}

			v.SetDefault(key, defaultValue)
			translationKeyMap[key] = v.GetString(key)
			return translationKeyMap[key]
		}, func() {
			// dump the translationKeyMap to a json file
			if err := DumpTranslationKeyMap(translationKeyMap); err != nil {
				log.Fatalf("Could not dump translation key map: %v", err)
			}
		}
}

// DumpTranslationKeyMap writes the translation map to a json file called github-mcp-server-config.json
// It preserves any existing filter_patterns configuration.
func DumpTranslationKeyMap(translationKeyMap map[string]string) error {
	// Read existing config to preserve filter_patterns
	v := viper.New()
	v.SetConfigName("github-mcp-server-config")
	v.SetConfigType("json")
	v.AddConfigPath(".")

	var existingFilterPatterns []string
	if err := v.ReadInConfig(); err == nil {
		// Config file exists, preserve filter_patterns if present
		if v.IsSet("filter_patterns") {
			existingFilterPatterns = v.GetStringSlice("filter_patterns")
		}
	}

	// Create output map with translations
	output := make(map[string]any)
	for k, v := range translationKeyMap {
		output[k] = v
	}

	// Add filter_patterns if they exist
	if len(existingFilterPatterns) > 0 {
		output["filter_patterns"] = existingFilterPatterns
	}

	file, err := os.Create("github-mcp-server-config.json")
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer func() { _ = file.Close() }()

	// marshal the map to json
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling map to JSON: %v", err)
	}

	// write the json data to the file
	if _, err := file.Write(jsonData); err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}

	return nil
}
