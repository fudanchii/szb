package sysstats

import (
	"fmt"
	"net"
	"strings"
	"time"
)

type NetworkStats struct {
	ifaceInfoBuffer string
}

func NewNetworkStats() *NetworkStats {
	nstats := &NetworkStats{}

	nstats.populateIfaceInfo()

	go func() {
		for {
			time.Sleep(30 * time.Second)

			nstats.populateIfaceInfo()
		}
	}()

	return nstats
}

func (nstats *NetworkStats) populateIfaceInfo() error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	ifaceList := []string{}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return err
		}

		if iface.Name == "lo" || len(addrs) == 0 {
			continue
		}

		addrsList := []string{}
		for _, addr := range addrs {
			if strings.HasPrefix(addr.String(), "fe80") {
				continue
			}
			addrsList = append(addrsList, addr.String())
		}
		ifaceList = append(ifaceList, fmt.Sprintf("%s ~ %s", iface.Name, strings.Join(addrsList, ", ")))
	}

	nstats.ifaceInfoBuffer = strings.Join(ifaceList, " | ")

	return nil
}

func (nstats *NetworkStats) String() string {
	return nstats.ifaceInfoBuffer
}
