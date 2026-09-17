/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"ssc/utils"
	"runtime"
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

	defaultSettings := defaultSettingsJSON()

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

// EnsureConfig initializes settings before commands such as init read them.
func EnsureConfig() {
	homedir, err := os.UserHomeDir()
	if err != nil { utils.Exit(err) }
	if err := ensureConfig(homedir + "/.sscconfig.json"); err != nil { utils.Exit(err) }
}

func ensureConfig(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) { return nil }
	if err != nil { return err }
	_, err = f.WriteString(defaultSettingsJSON())
	closeErr := f.Close()
	if err != nil { return err }
	return closeErr
}

func defaultSettingsJSON() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
		if runtime.GOOS == "windows" { editor = "notepad" }
	}
	settings := map[string]interface{}{
		"defaultBranch": "master", "aliases": map[string]string{},
		"commitMessagePrompt": "Input a commit message: ",
		"forceBranchDeletion": "false", "editor": editor,
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	return string(data) + "\n"
}
