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

	"github.com/IBM/netaddr"
	"github.com/sonnt85/gosutils/arp"
	"github.com/sonnt85/snetutils"

	//	"github.com/sonnt85/gosutils/sutils"
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"
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

func init() {
	diface, _ := snetutils.NetGetInterfaceInfo(snetutils.IfaceIname)
	cobra.OnInitialize(initConfig)
	arpCmd.Flags().DurationVarP(&durFlag, "duration", "d", 500*time.Millisecond, "timeout for ARP request")
	arpCmd.Flags().StringVarP(&ifaceFlag, "interface", "I", diface, "network interface to use for ARP request")
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

	ipset := new(netaddr.IPSet)

	if len(ipFlag) != 0 {
		netaddr.ParseIP(ipFlag)
		ipset.Insert(netaddr.ParseIP(ipFlag))
	} else {
		if ipnet, err := snetutils.NetGetCIDR(ifaceFlag); err == nil {
			fmt.Println(ipnet)

			if ipNet, err := netaddr.ParseCIDRToNet(ipnet); err == nil {
				ipset.InsertNet(ipNet)
			}
		}
	}
	cnt := 0
	cntF := 0
	fmt.Println("Arp on interface ", ifaceFlag)
	for _, ip := range ipset.GetIPs(65536) {
		// Request hardware address for IP address
		cnt++
		// Set request deadline from flag
		if err := c.SetDeadline(time.Now().Add(durFlag)); err != nil {
			continue
		}
		mac, err := c.Resolve(ip)
		if err != nil {
			cntF++
			//			fmt.Println(ip.String(), err)
			continue
		}
		fmt.Printf("%s -> %s\n", ip, mac)

	}
	fmt.Printf("False/Total %d/%d\n", cntF, cnt)

	return nil
}

func main() {
	if err := arpCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
