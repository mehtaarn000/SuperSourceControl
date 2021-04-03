package core

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"github.com/tidwall/sjson"
	"bufio"
)

// GetSetting gets the passed setting from the .sscconfig.json file in home directory
func GetSetting(setting string) string {
	// Get .sscconfig.json file from home directory
	homedir, err := os.UserHomeDir()
	get_settings, err := ioutil.ReadFile(homedir + "/.sscconfig.json")

	data := []byte(get_settings)

	// Unmarshal/parse data and store it in objmap
	var objmap map[string]interface{}
	if err := json.Unmarshal(data, &objmap); err != nil {
		panic(err)
	}

	// to parse setting
	value := objmap[setting]
	
	// map[value that doesn't exist] returns an empty string
	if value == "" || value == "\n" {
		panic("Setting '" + setting + "' doesn't exist.")
	}

	if err != nil {
		panic(err)
	}

	return value.(string)
}

// ChangeSetting changes a setting in the .sscconfig.json file in home directory
func ChangeSetting(setting string, new_setting string) {
	// Get .sscconfig.json file from home directory
	homedir, err := os.UserHomeDir()
	get_settings, err := ioutil.ReadFile(homedir + "/.sscconfig.json")

	// Create new settings
	newsettings, jsonerr := sjson.Set(string(get_settings), setting, new_setting)

	if jsonerr != nil {
		panic(err)
	}

	// Write new settings back to config file
	writer, err := os.Create(homedir + "/.sscconfig.json")
	writer.WriteString(newsettings)

	if err != nil {
		panic(err)
	}
}

func DefaultSettings(force bool) {
	// Get .sscconfig.json file from home directory
	homedir, err := os.UserHomeDir()

	// Aliases may be implemented in the future
	defaultSettings := `{
	"defaultBranch": "master",
	"aliases": {},
	"commitMessagePrompt": "Input a commit message: ",
	"forceBranchDeletion": "false"
}`
	
	if !force {
		scanner := bufio.NewScanner(os.Stdin)
		for {
			print("Are you sure you want to restore all settings to default [y/n]?")
			scanner.Scan()

			confirm = scanner.Text()
			if confirm == "Y" || confirm == "N" || confirm == "y" || confirm == "n" {
				break
			}
		}
		
		if confirm == "Y" || confirm == "y" {
			writer, err := os.Create(homedir + "/.sscconfig.json")
			writer.WriteString(defaultSettings)
			if err != nil {
				panic(err)
			}

		} else {
			return
		}
	}

	writer, err := os.Create(homedir + "/.sscconfig.json")
	writer.WriteString(defaultSettings)

	if err != nil {
		panic(err)
	}
}