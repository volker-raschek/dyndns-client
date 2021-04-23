package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"git.cryptic.systems/volker.raschek/dyndns-client/pkg/types"
	log "github.com/sirupsen/logrus"
)

//go:embed config.json
var defaultConfig string

// GetDefaultConfiguration returns a default configuration
func GetDefaultConfiguration() (*types.Config, error) {
	cnf := new(types.Config)
	jsonDecoder := json.NewDecoder(strings.NewReader(defaultConfig))

	err := jsonDecoder.Decode(cnf)
	if err != nil {
		return nil, fmt.Errorf("failed to decode default config: %w", err)
	}

	defaultInterface, err := getDefaultInterfaceByIP()
	if err != nil {
		return nil, err
	}
	cnf.Ifaces = []string{defaultInterface.Name}

	return cnf, nil
}

// Read config from a file
func Read(cnfFile string) (*types.Config, error) {

	// Load burned in configuration if config not available
	if _, err := os.Stat(cnfFile); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(cnfFile), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
		cnf, err := GetDefaultConfiguration()
		if err != nil {
			return nil, err
		}

		err = cnf.Validate()
		if err != nil {
			return nil, err
		}

		log.Infof("use embedded configuration")

		return cnf, nil
	}

	f, err := os.Open(cnfFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	cnf := new(types.Config)
	jsonDecoder := json.NewDecoder(f)
	err = jsonDecoder.Decode(cnf)
	if err != nil {
		return nil, fmt.Errorf("failed to decode json: %w", err)
	}

	for _, iface := range cnf.Ifaces {
		if _, err := net.InterfaceByName(iface); err != nil {
			return nil, fmt.Errorf("unknown interface: %v", iface)
		}
	}

	err = cnf.Validate()
	if err != nil {
		return nil, err
	}

	log.Infof("use configuration from file %v", cnfFile)

	return cnf, nil
}

// Write config into a file
func Write(cnf *types.Config, cnfFile string) error {
	if _, err := os.Stat(filepath.Dir(cnfFile)); os.IsNotExist(err) {
		err := os.MkdirAll(filepath.Dir(cnfFile), 0755)
		if err != nil {
			return err
		}
	}

	f, err := os.Create(cnfFile)
	if err != nil {
		return fmt.Errorf("failed to create file %v: %v", cnfFile, err)
	}
	defer f.Close()

	jsonEncoder := json.NewEncoder(f)
	jsonEncoder.SetIndent("", "  ")
	err = jsonEncoder.Encode(cnf)
	if err != nil {
		return fmt.Errorf("failed to encode json: %w", err)
	}
	return nil
}

func getDefaultInterfaceByIP() (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to fet network interfaces from kernel: %w", err)
	}

	defaultIP := getOutboundIP()

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, fmt.Errorf("failed to list ip addresses for interface %v: %w", iface.Name, err)
		}

		for _, addr := range addrs {
			addrIP := strings.Split(addr.String(), "/")[0]
			if addrIP == defaultIP.String() {
				return &iface, nil
			}
		}
	}

	return nil, fmt.Errorf("no interface found fo ip address %v", defaultIP)
}

func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}
