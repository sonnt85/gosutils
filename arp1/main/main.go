package main

import (
	"fmt"
	"github.com/sonnt85/gosutils/arp"
)

func main() {
	for ip, _ := range arp.Table() {
		log.Printf("%s : %s\n", ip, arp.Search(ip))
	}
}
