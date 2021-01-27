/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"

	"github.com/sonnt85/gosutils/arp"
	//	"github.com/sonnt85/gosutils/sutils"
	"github.com/spf13/cobra"
	"net"
	"os"
	"strings"
	"time"
)

var (
	// durFlag is used to set a timeout for an ARP request
	durFlag time.Duration

	// ifaceFlag is used to set a network interface for ARP requests
	ifaceFlag string

	// ipFlag is used to set an IPv4 address destination for an ARP request
	ipFlag string
)

// arpCmd represents the arp command
var arpCmd = &cobra.Command{
	Use:   "arp",
	Short: "Scan MAC, hostname",
	Long:  `Arp tool`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		return scan()
	},
}

func initConfig() {
	return
}

func GetOutboundIP() string {
	conn, err := net.DialTimeout("udp", "1.1.1.1:80", time.Second)
	if err != nil {
		log.Println(err)
		return ""
	} else {
		defer conn.Close()
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

func NetGetMacCIDR() (string, error) {
	ret := NetGetDefaultInterface(1)
	if ret != "" {
		return ret, nil
	} else {
		return "", nil
	}
}

//infotype 0 interface name, 1 macaddr, 2 cird, >2 lanip]
func NetGetDefaultInterface(infotype int) (info string) {
	// get all the system's or local machine's network interfaces
	LanIP := GetOutboundIP()
	interfaces, _ := net.Interfaces()
	for _, interf := range interfaces {

		if addrs, err := interf.Addrs(); err == nil {
			for _, addr := range addrs {
				if strings.Contains(addr.String(), LanIP) {
					if infotype == 0 {
						return interf.Name
					} else if infotype == 1 {
						return interf.HardwareAddr.String()
					} else if infotype == 2 {
						return addr.String()
					} else {
						return LanIP
					}
				}
			}
		}
	}
	return ""
}

func init() {
	cobra.OnInitialize(initConfig)
	arpCmd.Flags().DurationVarP(&durFlag, "duration", "d", 500*time.Millisecond, "timeout for ARP request")
	arpCmd.Flags().StringVarP(&ifaceFlag, "interface", "I", NetGetDefaultInterface(0), "network interface to use for ARP request")
	arpCmd.Flags().StringVarP(&ipFlag, "ip", "i", "", "IPv4 address destination for ARP request")
}

func scan() (err error) {

	// Ensure valid network interface
	ifi, err := net.InterfaceByName(ifaceFlag)
	if err != nil {
		return err
	}

	// Set up ARP client with socket
	c, err := arp.Dial(ifi)
	if err != nil {
		return err
	}
	defer c.Close()

	// Set request deadline from flag
	if err := c.SetDeadline(time.Now().Add(durFlag)); err != nil {
		return err
	}

	// Request hardware address for IP address
	ip := net.ParseIP(ipFlag).To4()
	mac, err := c.Resolve(ip)
	if err != nil {
		return err
	}

	log.Printf("%s -> %s", ip, mac)
	return nil
}

func main() {
	if err := arpCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
