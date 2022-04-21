package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
)

func main() {
	switch os.Args[1] {
	case "print-ipv6-trackers":
		printIpv6TcpTrackers()
	case "print-tracker-ips-location":
		printTrackerIpsAndCountries()
	}
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

// This consumes input from https://github.com/ngosang/trackerslist
func printTrackerIpsAndCountries() {
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
		for _, ip := range ips {
			loc, err := getIpLocation(ip)
			if err != nil {
				log.Printf("error getting location for ip %q: %v", ip, err)
				continue
			}
			fmt.Printf("%v\t%v\t%v\t%v\n", loc.Country, loc.City, ip, url)
		}
	}
}

type location struct {
	Country string
	City    string
}

func getIpLocation(ip net.IP) (loc location, err error) {
	resp, err := http.Get("http://go-geoserve.herokuapp.com/lookup/" + ip.String())
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("response status code: %v", resp.StatusCode)
		return
	}
	var dict interface{}
	err = json.NewDecoder(resp.Body).Decode(&dict)
	if err != nil {
		err = fmt.Errorf("decoding response json: %w", err)
		return
	}
	loc.Country = getEnName(dict, "Country")
	loc.City = getEnName(dict, "City")
	return
}

type jsonMap = map[string]interface{}

func getEnName(rootMap interface{}, rootKey string) string {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		log.Printf("error getting city from %v: %v", rootMap, r)
	}()
	return rootMap.(jsonMap)[rootKey].(jsonMap)["Names"].(jsonMap)["en"].(string)
}
