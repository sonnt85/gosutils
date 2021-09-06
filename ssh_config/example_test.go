package ssh_config_test

import (
	"fmt"
	"strings"

	"github.com/sonnt85/gosutils/ssh_config"
)

func ExampleHost_Matches() {
	pat, _ := ssh_config.NewPattern("test.*.example.com")
	host := &ssh_config.Host{Patterns: []*ssh_config.Pattern{pat}}
	log.Println(host.Matches("test.stage.example.com"))
	log.Println(host.Matches("othersubdomain.example.com"))
	// Output:
	// true
	// false
}

func ExamplePattern() {
	pat, _ := ssh_config.NewPattern("*")
	host := &ssh_config.Host{Patterns: []*ssh_config.Pattern{pat}}
	log.Println(host.Matches("test.stage.example.com"))
	log.Println(host.Matches("othersubdomain.any.any"))
	// Output:
	// true
	// true
}

func ExampleDecode() {
	var config = `
Host *.example.com
  Compression yes
`

	cfg, _ := ssh_config.Decode(strings.NewReader(config))
	val, _ := cfg.Get("test.example.com", "Compression")
	log.Println(val)
	// Output: yes
}

func ExampleDefault() {
	log.Println(ssh_config.Default("Port"))
	log.Println(ssh_config.Default("UnknownVar"))
	// Output:
	// 22
	//
}
