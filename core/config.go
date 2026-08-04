package core

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"github.com/tidwall/sjson"
)

// GetSetting gets the passed setting from the .sscconfig.json file in home directory
func GetSetting(setting string) string {
	// Get .sscconfig.json file from home directory
	homedir, err := os.UserHomeDir()
	get_settings, err := ioutil.ReadFile(homedir + "/.sscconfig.json")

	// Unmarshal/parse data and store it in objmap
	var objmap map[string]string
	if err := json.Unmarshal(get_settings, &objmap); err != nil {
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

	return value
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
	os.Create(homedir + "/.sscconfig.json")
	err = ioutil.WriteFile(homedir + "/.sscconfig.json", []byte(newsettings), os.FileMode(0777))

	if err != nil {
		panic(err)
	}
}