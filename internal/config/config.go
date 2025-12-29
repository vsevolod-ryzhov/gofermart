package config

import (
	"flag"
	"os"
	"time"
)

type Config struct {
	AppPort      string
	DatabaseURI  string
	AccrualPort  string
	JWTSecretKey string
	JWTTTL       time.Duration
}

func NewConfig() *Config {
	config := &Config{}

	var appPort string
	var databaseDSN string
	var accrualPort string

	flag.StringVar(&appPort, "a", "localhost:8080", "The address to bind the app to")
	flag.StringVar(&databaseDSN, "d", "", "Database connection string")
	flag.StringVar(&accrualPort, "r", "", "Accrual server port")

	flag.Parse()

	config.AppPort = appPort
	config.DatabaseURI = databaseDSN
	config.AccrualPort = accrualPort
	config.JWTTTL = 24 * time.Hour

	if envRunAddr, exists := os.LookupEnv("RUN_ADDRESS"); exists {
		config.AppPort = envRunAddr
	}
	if envDatabaseDSN, exists := os.LookupEnv("DATABASE_URI"); exists {
		config.DatabaseURI = envDatabaseDSN
	}
	if accrualRunAddr, exists := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); exists {
		config.AccrualPort = accrualRunAddr
	}
	if envKey, exists := os.LookupEnv("JWT_SECRET_KEY"); exists {
		config.JWTSecretKey = envKey
	}

	if config.JWTSecretKey == "" {
		// For passing project tests only
		config.JWTSecretKey = "ONzAJMacrrtieyP64OSuzR35YouGt5bD"
	}

	return config
}
