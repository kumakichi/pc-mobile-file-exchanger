package utils

import (
	"log"
	"net"
	"sort"
)

// GetIPs retrieves all IP addresses from network interfaces
func GetIPs() map[string]string {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatal(err)
	}
	ips := make(map[string]string, len(ifaces))

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			log.Printf("localAddresses: %+v\n", err.Error())
			continue
		}

		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ips[i.Name] = ipnet.IP.String()
				}
			}
		}
	}

	return ips
}

// Keys returns the keys of a map as a sorted slice
func Keys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
