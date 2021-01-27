package structtemplate

import (
	//	"fmt"
	"os"
	"strings"

	"github.com/sonnt85/gosutils/sutils"
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
}

func (app *AppInfo) UpdateInfo() {
	app.Mac = strings.Replace(sutils.NetGetMac(), ":", "", -1)
	app.Hostname, _ = os.Hostname()
	app.Uid = os.Geteuid()

	app.AppID = sutils.IDGet()
	app.Description = sutils.GetDescription()
}
