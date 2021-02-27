package main

import (
	"log"
	"net"
)

var ips map[string]string

func getIPs() map[string]string {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatal(err)
	}
	ips = make(map[string]string, len(ifaces))

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
