package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
)

type Config struct {
	DBUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

const configFileName = "/.youtube-custom-feeds-config.json"

func getConfigFilePath() (string, error) {
	path, err := os.UserHomeDir()
	if err != nil {
		newError := fmt.Sprintf("Error loacating HOME directory in getConfigFilePath(): %s", err)
		return "", errors.New(newError)
	}
	path = path + configFileName

	return path, nil
}

// Writes Config data to youtube-custom-feeds-config.json
func write(cfg Config) error {
	path, err := getConfigFilePath()
	if err != nil {
		newErr := fmt.Sprintf("Error in write() - issue getting config file path: %s", err)
		return errors.New(newErr)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		newErr := fmt.Sprintf("Error in write() - issue Marshaling Config: %s", err)
		return errors.New(newErr)
	}

	err = os.WriteFile(path, data, 0666)
	if err != nil {
		newErr := fmt.Sprintf("Error in write() - issue writing data to youtube-custom-feeds-config.json: %s", err)
		return errors.New(newErr)
	}

	return nil
}

// Reads the data from youtube-custom-feeds-config.json and
// stores it in a Config struct.
func Read() Config {
	path, err := getConfigFilePath()
	if err != nil {
		log.Fatalf("Error getting file path in Read(): %s", err)
	}
	configFile, err := os.Open(path)
	if err != nil {
		log.Fatalf("Error in Read() - Issue opening config json: %s", err)
	}

	var configStruct Config
	decoder := json.NewDecoder(configFile)
	if err := decoder.Decode(&configStruct); err != nil {
		log.Fatalf("Error in Read() - Issue decoding config json: %s", err)
	}

	return configStruct
}

// Sets the CurrentUserName of the Config struct
// to the provided userName.
func (cfg Config) SetUser(userName string) {
	cfg.CurrentUserName = userName
	if err := write(cfg); err != nil {
		log.Fatalf("Error in SetUser(): %s", err)
	}
}
