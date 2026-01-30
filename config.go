package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

const usage = `Usage:
  -b, --bind           address:port to run the server on (default: 0.0.0.0:3001)
  -c, --config         Path to a configuration file (default: config.toml)
  -d, --database       Path to SQLite database file (default: ./pastes.db)
  -s, --serve-path     Path to serve pastes from (default: /p/)

Environment Variables:
  PB_BIND              Same as --bind
  PB_DATABASE_PATH     Same as --database
  PB_DEBUG             Set to "true" to enable debug mode
  PB_SERVE_PATH        Same as --serve-path`

// Default config
func defaultConfig() Config {
	return Config{
		Bind:         "0.0.0.0:3001",
		ServePath:    "/p/",
		DatabasePath: "./pastes.db",
	}
}

func GenerateConfig() Config {
	var bindOpt string
	var configFile string
	var configFileSet bool
	var databasePathOpt string
	var debugOpt bool
	var servePathOpt string

	flag.StringVar(&bindOpt, "b", "", "address:port to run the server on")
	flag.StringVar(&bindOpt, "bind", "", "address:port to run the server on")
	flag.StringVar(&configFile, "c", "", "Path to the configuration file")
	flag.StringVar(&configFile, "config", "", "Path to the configuration file")
	flag.StringVar(&databasePathOpt, "d", "", "Path to SQLite database file")
	flag.StringVar(&databasePathOpt, "database", "", "Path to SQLite database file")
	flag.BoolVar(&debugOpt, "debug", false, "enable debug mode")
	flag.StringVar(&servePathOpt, "s", "", "Path to serve pastes from")
	flag.StringVar(&servePathOpt, "serve-path", "", "Path to serve pastes from")

	flag.Usage = func() {
		fmt.Println(usage)
	}

	flag.Parse()

	// Check if a config file was specified
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "config" || f.Name == "c" {
			configFileSet = true
		}
	})

	if configFile == "" {
		configFile = "config.toml"
	}

	// Check if the config file exists
	var config Config
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if configFileSet {
			log.Fatalf("Config file %v specified but not found.\n", configFile)
		}
		fmt.Printf("Config file %v not found. Using defaults.\n", configFile)
		config = defaultConfig()
	} else if err != nil {
		log.Fatalf("Error accessing config file %v: %v\n", configFile, err)
	} else {
		// Load the config file
		fmt.Printf("Loading config from %v\n", configFile)
		config = loadConfig(configFile)
	}

	// Override with environment variables (if set)
	if envBind := os.Getenv("PB_BIND"); envBind != "" {
		config.Bind = envBind
	}
	if envDB := os.Getenv("PB_DATABASE_PATH"); envDB != "" {
		config.DatabasePath = envDB
	}
	if envDebug := os.Getenv("PB_DEBUG"); envDebug == "true" || envDebug == "1" {
		config.Debug = true
	}
	if envServePath := os.Getenv("PB_SERVE_PATH"); envServePath != "" {
		config.ServePath = envServePath
	}

	// Override the config values with the command-line flags (highest priority)
	options := map[*string]*string{
		&bindOpt:         &config.Bind,
		&databasePathOpt: &config.DatabasePath,
		&servePathOpt:    &config.ServePath,
	}

	for option, configField := range options {
		if *option != "" {
			*configField = *option
		}
	}

	if debugOpt {
		config.Debug = true
	}

	return config
}

func loadConfig(configFile string) Config {
	config := defaultConfig()

	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Fatal(err)
	}

	return config
}
