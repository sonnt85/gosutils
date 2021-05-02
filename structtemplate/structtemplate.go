package structtemplate

import (
	//	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/sonnt85/gosutils/sutils"
	"github.com/sonnt85/snetutils"
)

type AppInfo struct {
	Description   string `json:"des"`
	Mac           string `json:"mac"`
	RemoteVNCPort int    `json:"vncp"`
	RemoteVNCIP   string `json:"vncip"`
	RemoteSSHPort int    `json:"sshp"`
	RemoteSSHIP   string `json:"sship"`
	Hostname      string `json:"hostname"`
	Uid           int    `json:"uid"`
	AppID         string `json:"appid"`
	Os            string `json: "os"`
	Arch          string `json :"arch"`
	LastUpdate    time.Time
}

func (app *AppInfo) GetArchExtension() string {
	if app.Arch == "windows" {
		return app.Arch + ".exe"
	} else {
		return app.Arch
	}
}
func (app *AppInfo) UpdateInfo() {
	app.Os = runtime.GOOS
	app.Arch = runtime.GOARCH
	mac := snetutils.NetGetStaticMac()
	if len(mac) == 0 {
		if mact, err := snetutils.NetGetMac(); err == nil {
			mac = mact
		}
	}
	mac = strings.Replace(mac, ":", "", -1)
	if len(mac) != 0 {
		app.Mac = mac
	}
	app.Hostname, _ = os.Hostname()
	app.Uid = os.Geteuid()

	app.AppID = sutils.IDGet()
	app.Description = sutils.GetDescription()
}
