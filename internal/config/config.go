package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
)

const GATOR_CONFIG_FILE = ".gatorconfig.json" 

type Config struct {
	DB_URL            string `json:"db_url"`
	CURRENT_USER_NAME string `json:"current_user_name"`
}

func (c *Config) Read() {
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Encountered an error: ", err)
	}
	config_path := path.Join(homedir, GATOR_CONFIG_FILE)
	data, err := os.ReadFile(config_path)
	if err != nil {
		log.Fatal("Error reading config file: ", err)
	}
	err = json.Unmarshal(data, c)
	if err != nil {
		log.Fatal("Error reading config file: ", err)
	}
}

func (c *Config) SetUser(username string) error {
    c.CURRENT_USER_NAME = username

	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Encountered an error: ", err)
	}
	config_path := path.Join(homedir, GATOR_CONFIG_FILE)
	
	data, err := json.Marshal(c)
	data = append(data, '\n')
	if err != nil {
		return fmt.Errorf("could not set user: %w", err)
	}

	err = os.WriteFile(config_path, data, os.ModeExclusive)
	if err != nil {
		return fmt.Errorf("could not set user: %w", err)
	}

	return nil
}
