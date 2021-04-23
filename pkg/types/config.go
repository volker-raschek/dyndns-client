package types

import (
	"fmt"

	"github.com/asaskevich/govalidator"
)

type Config struct {
	Ifaces   []string            `json:"interfaces"`
	Zones    map[string]*Zone    `json:"zones"`
	TSIGKeys map[string]*TSIGKey `json:"tsig-keys"`
}

func (c *Config) Validate() error {
	if len(c.Zones) <= 0 {
		return fmt.Errorf("no dns zones configured")
	}

	if len(c.Ifaces) <= 0 {
		return fmt.Errorf("no interfaces configured")
	}

ZONE_LOOP:
	for _, zone := range c.Zones {
		if !govalidator.IsIP(zone.DNSServer) && !govalidator.IsDNSName(zone.DNSServer) {
			return fmt.Errorf("invalid dns server %v", zone.DNSServer)
		}

		if !govalidator.IsDNSName(zone.Name) {
			return fmt.Errorf("invalid dns zone name %v", zone.Name)
		}

		for _, tsigkey := range c.TSIGKeys {
			if tsigkey.Name == zone.TSIGKeyName {
				continue ZONE_LOOP
			}
		}
		return fmt.Errorf("no matching tsigkey found for zone %v", zone.Name)
	}
	return nil
}
