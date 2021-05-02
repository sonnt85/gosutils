package main

import (
	"bytes"
	"flag"
	"io"
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/sonnt85/gosutils/arp"
	"github.com/sonnt85/gosutils/ethernet"
	"github.com/sonnt85/snetutils"
)

var (
	// ipFlag is used to set an IPv4 address to proxy ARP on behalf of
	ipFlag = flag.String("ip", "", "IP address for device to proxy ARP on behalf of")
)

func main() {
	// ifaceFlag is used to set a network interface for ARP traffic
	diface := "eth0"
	if iface, err := snetutils.NetGetInterfaceInfo(snetutils.IfaceIname); err == nil {
		diface = iface
	}
	ifaceFlag := flag.String("i", diface, "network interface to use for ARP traffic")

	fakeMac := flag.String("m", "", "Mac add respone for all ARP request")

	flag.Parse()
	var macRespone net.HardwareAddr
	if len(*fakeMac) != 0 {
		macRespone, _ = net.ParseMAC(*fakeMac)
	}
	if len(*ipFlag) == 0 {
		*ipFlag, _ = snetutils.NetGetInterfaceIpv4Addr(*ifaceFlag)
	}
	// Ensure valid interface and IPv4 address
	ifi, err := net.InterfaceByName(*ifaceFlag)
	if err != nil {
		log.Fatal(err)
	}
	ip := net.ParseIP(*ipFlag).To4()
	if ip == nil {
		log.Fatalf("invalid IPv4 address: %q", *ipFlag)
	}

	client, err := arp.Dial(ifi)
	if err != nil {
		log.Fatalf("couldn't create ARP client: %s", err)
	}

	// Handle ARP requests bound for designated IPv4 address, using proxy ARP
	// to indicate that the address belongs to this machine
	for {
		pkt, eth, err := client.Read()
		if err != nil {
			if err == io.EOF {
				log.Println("EOF")
				break
			}
			log.Fatalf("error processing ARP requests: %s", err)
		}

		// Ignore ARP replies
		if pkt.Operation != arp.OperationRequest {
			continue
		}

		// Ignore ARP requests which are not broadcast or bound directly for
		// this machine
		if !bytes.Equal(eth.Destination, ethernet.Broadcast) && !bytes.Equal(eth.Destination, ifi.HardwareAddr) {
			continue
		}

		log.Printf("request: who-has %s?  tell %s (%s)", pkt.TargetIP, pkt.SenderIP, pkt.SenderHardwareAddr)
		if len(macRespone) != 0 && !pkt.TargetIP.Equal(ip) {
			log.Infof("Fake mac %s for %s", macRespone.String(), pkt.TargetIP.String())
			for i := 0; i < 1000000; i++ {
				//				pkt.SenderHardwareAddr = macRespone
				if err := client.Reply(pkt, macRespone, pkt.TargetIP); err != nil {
					log.Fatal(err)
				}
			}
			continue
		}
		// Ignore ARP requests which do not indicate the target IP
		if !pkt.TargetIP.Equal(ip) {
			continue
		}

		log.Printf("  reply: %s is-at %s", ip, ifi.HardwareAddr)
		if err := client.Reply(pkt, ifi.HardwareAddr, ip); err != nil {
			log.Fatal(err)
		}
	}
}
