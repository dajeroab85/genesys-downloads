package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	GenesysCloudEnvironment string
	ClientID                string
	ClientSecret            string
	MaxDownloadWorkers      int
	PollRetries             int
	PollInterval            time.Duration
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: .env file not found. Using system environment variables.")
	}

	maxWorkers, err := strconv.Atoi(os.Getenv("MAX_DOWNLOAD_WORKERS"))
	if err != nil {
		maxWorkers = 10 // default
	}

	retries, err := strconv.Atoi(os.Getenv("POLL_RETRIES"))
	if err != nil {
		retries = 50 // default
	}

	interval, err := strconv.Atoi(os.Getenv("POLL_INTERVAL"))
	if err != nil {
		interval = 25 // default seconds
	}

	cfg := &Config{
		GenesysCloudEnvironment: os.Getenv("GENESYS_ENVIRONMENT"),
		ClientID:                os.Getenv("CLIENT_ID"),
		ClientSecret:            os.Getenv("CLIENT_SECRET"),
		MaxDownloadWorkers:      maxWorkers,
		PollRetries:             retries,
		PollInterval:            time.Duration(interval) * time.Second,
	}

	return cfg, nil
}
