package structtemplate

type AppInfo struct {
	Description   string `json : "des"`
	Mac           string `json : "mac"`
	RemoteVNCPort int    `json : "sshp"`
	RemoteSSHPort int    `json : "vncp"`
	Hostname      string `json : "hostname"`
	Uid           int    `json : "uid"`
	AppID         string `json : "appid"`
}
