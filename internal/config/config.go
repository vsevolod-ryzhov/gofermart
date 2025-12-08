package config

import (
	"flag"
	"os"
)

type Config struct {
	AppPort     string
	DatabaseDSN string
}

func NewConfig() *Config {
	config := &Config{}

	var appPort string
	var databaseDSN string

	flag.StringVar(&appPort, "a", "localhost:8080", "The address to bind the app to")
	flag.StringVar(&databaseDSN, "d", "", "Database connection string")

	flag.Parse()

	config.AppPort = appPort
	config.DatabaseDSN = databaseDSN

	if envRunAddr, exists := os.LookupEnv("RUN_ADDRESS"); exists {
		config.AppPort = envRunAddr
	}
	if envDatabaseDSN, exists := os.LookupEnv("DATABASE_DSN"); exists {
		config.DatabaseDSN = envDatabaseDSN
	}

	return config
}
