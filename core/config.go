package core

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"ssc/utils"

	"github.com/tidwall/sjson"
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
		utils.Exit(err)
	}

	// to parse setting
	value := objmap[setting]

	// map[value that doesn't exist] returns an empty string
	if value == "" || value == "\n" {
		utils.Exit("Setting '" + setting + "' doesn't exist.")
	}

	if err != nil {
		utils.Exit(err)
	}

	return value.(string)
}

// ChangeSetting changes a setting in the .sscconfig.json file in home directory
func ChangeSetting(setting string, new_setting string) {

	// If the user changes the default branch setting, validate the new branch name
	if setting == "defaultBranch" {
		if !validateBranchName(new_setting) {
			utils.Exit("Invalid branch name: '" + new_setting + "'")
		}

	}

	// Get .sscconfig.json file from home directory
	homedir, err := os.UserHomeDir()
	get_settings, err := ioutil.ReadFile(homedir + "/.sscconfig.json")

	// Create new settings
	newsettings, jsonerr := sjson.Set(string(get_settings), setting, new_setting)

	if jsonerr != nil {
		utils.Exit(err)
	}

	// Write new settings back to config file
	writer, err := os.Create(homedir + "/.sscconfig.json")
	writer.WriteString(newsettings)

	if err != nil {
		utils.Exit(err)
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
				utils.Exit(err)
			}

		} else {
			return
		}
	}

	writer, err := os.Create(homedir + "/.sscconfig.json")
	writer.WriteString(defaultSettings)

	if err != nil {
		utils.Exit(err)
	}
}
