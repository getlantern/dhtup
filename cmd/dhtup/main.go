package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
)

func main() {
	printIpv6TcpTrackers()
}

// This consumes input from https://github.com/ngosang/trackerslist
func printIpv6TcpTrackers() {
	scanner := bufio.NewScanner(os.Stdin)
	//trackers:
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			continue
		}
		//log.Printf("processing %q", text)
		url, err := url.Parse(text)
		if err != nil {
			log.Printf("parsing %q as url: %v", text, err)
			continue
		}
		host := url.Hostname()
		ips, err := net.LookupIP(host)
		if err != nil {
			log.Printf("looking up ip for %q: %v", host, err)
			continue
		}
		has6 := false
		for _, ip := range ips {
			if ip.To4() != nil {
				// I didn't find any IPv6-only trackers so I had to relax the constraints.
				//continue trackers
			} else {
				has6 = true
			}
		}
		if !has6 {
			continue
		}
		fmt.Printf("%v: %v\n", url, ips)
	}
}
