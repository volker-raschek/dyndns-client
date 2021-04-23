package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"git.cryptic.systems/volker.raschek/dyndns-client/pkg/types"
	"git.cryptic.systems/volker.raschek/dyndns-client/pkg/updater"
	"github.com/asaskevich/govalidator"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func Start(cnf *types.Config) {
	addrUpdates := make(chan netlink.AddrUpdate, 1)
	done := make(chan struct{}, 1)
	err := netlink.AddrSubscribeWithOptions(addrUpdates, done, netlink.AddrSubscribeOptions{
		ListExisting: true,
	})
	if err != nil {
		log.Fatalf("failed to subscribe netlink notifications from kernel: %v", err.Error())
	}

	interuptChannel := make(chan os.Signal, 1)
	signal.Notify(interuptChannel, syscall.SIGINT, syscall.SIGTERM)

	ctx := context.Background()
	daemonCtx, cancle := context.WithCancel(ctx)
	defer cancle()

	updaters, err := getUpdaterForEachZone(cnf)
	if err != nil {
		log.Fatalf("%v", err.Error())
	}

	if err := pruneRecords(daemonCtx, updaters, cnf.Zones); err != nil {
		log.Fatalf("%v", err.Error())
	}

	for {
		interfaces, err := netlink.LinkList()
		if err != nil {
			log.Fatal("%v", err.Error())
		}

		select {
		case update := <-addrUpdates:

			interfaceLogger := log.WithFields(log.Fields{
				"ip": update.LinkAddress.IP.String(),
			})

			// search interface by index
			iface, err := searchInterfaceByIndex(update.LinkIndex, interfaces)
			if err != nil {
				log.Errorf("%v", err.Error())
				continue
			}
			interfaceLogger = interfaceLogger.WithField("device", iface.Attrs().Name)

			var recordType string
			switch {
			case govalidator.IsIPv4(strings.TrimRight(update.LinkAddress.IP.String(), "/")):
				recordType = "A"
			case govalidator.IsIPv6(strings.TrimRight(update.LinkAddress.IP.String(), "/")):
				recordType = "AAAA"
			default:
				interfaceLogger.Error("failed to detect record type")
				continue
			}
			interfaceLogger = interfaceLogger.WithField("rr", recordType)

			interfaceLogger.Debug("receive kernel notification for interface")

			// filter out not configured interfaces
			if !matchInterfaces(iface.Attrs().Name, cnf.Ifaces) {
				interfaceLogger.Warn("interface is not part of the allowed interface list")
				continue
			}

			// filter out notification for a bad interface ip address, for example link-local-addresses
			if update.LinkAddress.IP.IsLoopback() || strings.HasPrefix(update.LinkAddress.IP.String(), "fe80") {
				interfaceLogger.Warn("interface is a loopback device or part of a loopback network")
				continue
			}

			// decide if trigger a add or delete event
			if update.NewAddr {
				err = addIPRecords(daemonCtx, interfaceLogger, updaters, cnf.Zones, recordType, update.LinkAddress.IP)
				if err != nil {
					interfaceLogger.Error(err.Error())
				}
			} else {
				err = removeIPRecords(daemonCtx, interfaceLogger, updaters, cnf.Zones, recordType, update.LinkAddress.IP)
				if err != nil {
					interfaceLogger.Error(err.Error())
				}
			}

		case killSignal := <-interuptChannel:
			log.Debugf("got signal: %v", killSignal)
			log.Debugf("daemon was killed by: %v", killSignal)
			return
		}
	}
}

func getUpdaterForEachZone(config *types.Config) (map[string]updater.Updater, error) {
	updaterCollection := make(map[string]updater.Updater)

	for zoneName, zone := range config.Zones {
		nsUpdater, err := updater.NewNSUpdate(zone.DNSServer, config.TSIGKeys[zone.TSIGKeyName])
		if err != nil {
			return nil, err
		}
		updaterCollection[zoneName] = nsUpdater
	}

	return updaterCollection, nil
}

func matchInterfaces(iface string, ifaces []string) bool {
	for _, i := range ifaces {
		if i == iface {
			return true
		}
	}
	return false
}

func searchInterfaceByIndex(index int, interfaces []netlink.Link) (netlink.Link, error) {
	for _, iface := range interfaces {
		if iface.Attrs().Index == index {
			return iface, nil
		}
	}
	return nil, fmt.Errorf("can not find interface by index %v", index)
}

func addIPRecords(ctx context.Context, logEntry *log.Entry, updaters map[string]updater.Updater, zones map[string]*types.Zone, recordType string, ip net.IP) error {

	var (
		errorChannel = make(chan error, len(zones))
		wg           = new(sync.WaitGroup)
	)

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get host name from kernel: %w", err)
	}
	hostname = strings.ToLower(hostname)

	if !verifyHostname(hostname) {
		return fmt.Errorf("host name not valid: %w", err)
	}

	for zoneName := range zones {
		wg.Add(1)

		go func(ctx context.Context, zoneName string, hostname string, recordType string, ip net.IP, wg *sync.WaitGroup) {

			zoneLogger := logEntry.WithFields(log.Fields{
				"zone":     zoneName,
				"hostname": hostname,
			})

			defer wg.Done()

			pruneRecordCtx, cancle := context.WithTimeout(ctx, time.Second*15)
			defer cancle()

			fqdn := fmt.Sprintf("%v.%v", hostname, zoneName)

			err := updaters[zoneName].AddRecord(pruneRecordCtx, fqdn, 60, recordType, ip.String())
			if err != nil {
				errorChannel <- fmt.Errorf("failed to remove record type %v for %v: %v", recordType, fqdn, err.Error())
				return
			}

			zoneLogger.Info("dns-record successfully updated")

		}(ctx, zoneName, hostname, recordType, ip, wg)
	}

	wg.Wait()
	close(errorChannel)

	for err := range errorChannel {
		if err != nil {
			return err
		}
	}

	return nil
}

func pruneRecords(ctx context.Context, updaters map[string]updater.Updater, zones map[string]*types.Zone) error {

	var (
		errorChannel = make(chan error, len(zones))
		wg           = new(sync.WaitGroup)
	)

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get host name from kernel: %w", err)
	}
	hostname = strings.ToLower(hostname)

	if !verifyHostname(hostname) {
		return fmt.Errorf("host name not valid: %w", err)
	}

	for zoneName := range zones {
		wg.Add(1)

		go func(zoneName string, hostname string, errorChannel chan<- error, wg *sync.WaitGroup) {
			defer wg.Done()

			pruneRecordCtx, cancle := context.WithTimeout(ctx, time.Second*15)
			defer cancle()

			fqdn := fmt.Sprintf("%v.%v", hostname, zoneName)

			err := updaters[zoneName].PruneRecords(pruneRecordCtx, fqdn)
			if err != nil {
				errorChannel <- fmt.Errorf("failed to prune %v: %v", fqdn, err)
				return
			}
		}(zoneName, hostname, errorChannel, wg)
	}

	wg.Wait()
	close(errorChannel)

	for err := range errorChannel {
		if err != nil {
			return err
		}
	}

	return nil
}

func removeIPRecords(ctx context.Context, logEntry *log.Entry, updaters map[string]updater.Updater, zones map[string]*types.Zone, recordType string, ip net.IP) error {

	var (
		errorChannel = make(chan error, len(zones))
		wg           = new(sync.WaitGroup)
	)

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get host name from kernel: %w", err)
	}
	hostname = strings.ToLower(hostname)

	if !verifyHostname(hostname) {
		return fmt.Errorf("host name not valid: %w", err)
	}

	for zoneName := range zones {
		wg.Add(1)

		go func(ctx context.Context, zoneName string, hostname string, recordType string, wg *sync.WaitGroup) {
			defer wg.Done()

			zoneLogger := logEntry.WithFields(log.Fields{
				"zone":     zoneName,
				"hostname": hostname,
			})

			pruneRecordCtx, cancle := context.WithTimeout(ctx, time.Second*15)
			defer cancle()

			fqdn := fmt.Sprintf("%v.%v", hostname, zoneName)

			err := updaters[zoneName].DeleteRecord(pruneRecordCtx, fqdn, recordType)
			if err != nil {
				errorChannel <- fmt.Errorf("failed to remove record type %v for %v: %v", recordType, fqdn, err.Error())
				return
			}

			zoneLogger.Info("dns-record successfully removed")

		}(ctx, zoneName, hostname, recordType, wg)
	}

	wg.Wait()
	close(errorChannel)

	for err := range errorChannel {
		if err != nil {
			return err
		}
	}

	return nil
}

// verifyHostname returns a boolean if the hostname id valid. The hostname does
// not contains any dot or local, localhost, localdomain.
func verifyHostname(hostname string) bool {

	if !validHostname.MatchString(hostname) {
		return false
	}

	hostnames := []string{
		"local",
		"localhost",
		"localdomain",
		"orbisos",
	}

	for i := range hostnames {
		if hostnames[i] == hostname {
			return false
		}
	}

	return true
}

var (
	validHostname = regexp.MustCompile(`^[a-zA-Z0-9]+([\-][a-zA-Z0-9]+)*$`)
)
