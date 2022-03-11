package main

import (
	"flag"
	"os"

	log "github.com/sirupsen/logrus"

	"git.cryptic.systems/volker.raschek/dyndns-client/pkg/config"
	"git.cryptic.systems/volker.raschek/dyndns-client/pkg/daemon"
)

var (
	version    string
	configPath string
)

func main() {
	switch os.Getenv("DYNDNS_CLIENT_LOGGER_LEVEL") {
	case "DEBUG", "debug":
		log.SetLevel(log.DebugLevel)
	case "WARN", "warn":
		log.SetLevel(log.WarnLevel)
	case "ERROR", "error":
		log.SetLevel(log.ErrorLevel)
	case "FATAL", "fatal":
		log.SetLevel(log.FatalLevel)
	case "INFO", "info":
		fallthrough
	default:
		log.SetLevel(log.InfoLevel)
	}

	switch os.Getenv("DYNDNS_CLIENT_LOGGER_FORMATTER") {
	case "JSON", "json":
		log.SetFormatter(&log.JSONFormatter{})
	case "TEXT", "text":
		fallthrough
	default:
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp: true,
		})
	}

	flag.StringVar(&configPath, "config", "/etc/dyndns-client/config.json", "Path to json config")

	log.Infof("version %v", version)

	cnf, err := config.Read(configPath)
	if err != nil {
		log.Fatal(err.Error())
	}

	daemon.Start(cnf)
}
