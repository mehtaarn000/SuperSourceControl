/* Copyright © 2021
Author : mehtaarn000
Email : arnavm834@gmail.com
*/

package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"ssc/utils"
	"strings"
)

// configPath permits isolated configuration through SSC_CONFIG_FILE.
func configPath() (string, error) {
	if path := os.Getenv("SSC_CONFIG_FILE"); path != "" {
		return path, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".sscconfig.json"), nil
}

func readSettings() (map[string]interface{}, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, fmt.Errorf("configuration must be a JSON object")
	}
	return settings, nil
}

func GetSetting(setting string) string {
	settings, err := readSettings()
	if err != nil {
		utils.Exit(err)
	}
	value, ok := settings[setting].(string)
	if !ok {
		utils.Exit(fmt.Errorf("setting %q is missing or is not a string", setting))
	}
	return value
}

// ConfiguredAuthor requires explicit identity; legacy configs need no migration.
func ConfiguredAuthor() (string, string, error) {
	settings, err := readSettings()
	if err != nil {
		return "", "", err
	}
	name, _ := settings["authorName"].(string)
	email, _ := settings["authorEmail"].(string)
	name, email = strings.TrimSpace(name), strings.TrimSpace(email)
	if err := validateIdentity(name, email); err != nil {
		return "", "", err
	}
	return name, email, nil
}

func ChangeSetting(setting string, value string) {
	if setting == "defaultBranch" && !validateBranchName(value) {
		utils.Exit("Invalid branch name")
	}
	if (setting == "authorName" || setting == "authorEmail") && (strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n\x00")) {
		utils.Exit("Author settings must be nonempty single-line strings")
	}
	settings, err := readSettings()
	if err != nil {
		utils.Exit(err)
	}
	settings[setting] = value
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		utils.Exit(err)
	}
	path, err := configPath()
	if err != nil {
		utils.Exit(err)
	}
	if err := ioutil.WriteFile(path, append(data, '\n'), 0644); err != nil {
		utils.Exit(err)
	}
}

func DefaultSettings(force bool) {
	path, err := configPath()
	if err != nil {
		utils.Exit(err)
	}
	if !force {
		print("Are you sure you want to restore all settings to default [y/n]?")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() || !strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
			return
		}
	}
	if err := ioutil.WriteFile(path, []byte(defaultSettingsJSON()), 0644); err != nil {
		utils.Exit(err)
	}
}

// EnsureConfig initializes settings before commands such as init read them.
func EnsureConfig() {
	path, err := configPath()
	if err != nil {
		utils.Exit(err)
	}
	if err := ensureConfig(path); err != nil {
		utils.Exit(err)
	}
}

func ensureConfig(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = f.WriteString(defaultSettingsJSON())
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func defaultSettingsJSON() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
		if runtime.GOOS == "windows" {
			editor = "notepad"
		}
	}
	settings := map[string]interface{}{
		"authorName": "", "authorEmail": "",
		"defaultBranch": "master", "aliases": map[string]string{},
		"commitMessagePrompt": "Input a commit message: ",
		"forceBranchDeletion": "false", "editor": editor,
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	return string(data) + "\n"
}
