package goat

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sonnt85/gonmmm"
	"github.com/sonnt85/gosutils/gogrep"
	"github.com/sonnt85/gosutils/sexec"
	"github.com/sonnt85/gosutils/sregexp"
	"github.com/sonnt85/gosutils/sutils"
	"github.com/sonnt85/gosystem"
	"github.com/tarm/serial"
)

var findATPort = false
var baud = 57600

func GetTTyAt(dev string) string {
	devat := dev
	if !findATPort {
		return devat
	}
	//	return devat
	if retregex := sregexp.New(`(^.+)([0-9]+)$`).FindStringSubmatch(dev); len(retregex) != 0 {
		index, _ := strconv.Atoi(retregex[2])
		index = index - 2
		atpath := fmt.Sprintf("%s%d", retregex[1], index)
		if _, err := os.Stat(atpath); !os.IsNotExist(err) {
			//			fmt.Printf("AT use new device %s\n", atpath)
			devat = atpath
		}
	}
	return devat
}

func ConfigAutoPort(b bool) {
	findATPort = b
}

func ConfigApn(dev, apn, username, password string) (err error) {
	fmt.Println("Configure apn for", dev)
	PDPdEL(dev, []int{0, 1, 2, 3, 4})
	if _, err := SetApn(dev, apn); err != nil {
		log.Errorf("Can not config apn %s %+v\n", apn, err)
		//		return err
	}
	//		nmcli con mod gsm${devbase} ipv4.dns "1.1.1.1 8.8.8.8 208.67.222.222"`,

	cmd := fmt.Sprintf(`dev=%s;
	devbase=${dev##*/};
	iface="${devbase}"
	apn=%s
    username=%s
    passord=%s
#    [[ $username == sora ]] && {
#	   ifname='*'
#	}
	nmcli connection add type gsm ifname "${iface}" con-name gsm${devbase} apn "${apn}" user "${username}" password "${passord}";
	nmcli con mod gsm${devbase} ipv4.dns "1.1.1.1"`,
		dev, apn, username, password)
	if _, _, err = sexec.ExecCommandShell(cmd, time.Second*1); err != nil {
		fmt.Println("Can not add  gsm connection")
	}

	time.Sleep(time.Second * 3)
	if pdps, err := MMGetPDP(); err == nil {
		fmt.Printf("Apn new configured: %v\n", pdps)
	} else {
		fmt.Printf("Can not get apn new configured for %s\n", dev)
	}

	if _, _, err = sexec.ExecCommandShell(fmt.Sprintf(`dev=%s;devbase=${dev##*/};dev status | grep gsm${devbase}| grep connected`, dev), time.Second*1); err != nil {
		fmt.Println("Can not connect to internet from USB LTE")
		return err
	}

	return nil
}

func MMConfigApn(dev, apn, username, password string) (err error) {
	log.Println("Configure apn for ", dev)
	iface := dev
	conName := "gsm" + dev
	//nmcli con mod gsm${devbase} ipv4.dns "1.1.1.1 8.8.8.8 208.67.222.222"`,
	//nmcli con mod gsm${devbase} ipv4.dns "1.1.1.1"
	//nmcli connection add type gsm ifname "*" con-name gsmttyUSB2 apn soracom.io user sora password sora

	if username == "" && gonmmm.NMConIsExist(conName) && gonmmm.NMConGetField(conName, "gsm.password") != "" {
		log.Warnf(`Delete old gsm connections "%s" "%s"`, conName, gonmmm.NMConGetField(conName, "gsm.password"))
		gonmmm.NMDelCon(conName)
	}

	if err := gonmmm.NMCreateConnection(conName, iface, "gsm", fmt.Sprintf(`apn "%s" user "%s" password "%s" ipv4.dns 1.1.1.1 connection.autoconnect yes`, apn, username, password)); err != nil {
		log.Error("can not add  gsm connection: ", err)
		return err
	}

	mapcon := map[string]string{"gsm.apn": apn, "gsm.password": password, "gsm.username": username, "ipv4.dns": "1.1.1.1"}
	for k, v := range mapcon {
		if !gonmmm.NMConEditFieldIfChange(conName, "gsm.password", password) {
			return fmt.Errorf("can not modify field %s to %s", k, v)
		}
	}

	return nil

	time.Sleep(time.Second * 10)
	if err := gonmmm.NMEnableCon(conName); err == nil {
		return nil
	} else {
		// log.Warn(err.Error())
		return err
	}

	time.Sleep(time.Second * 10)

	if !MMPDPIsConfigured(apn) {
		MMPDPdEL([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
		if errlist, err := MMSetApn(apn); err != nil {
			log.Errorf("Can not config apn %s:\n%s\n", apn, errlist)
			return fmt.Errorf("Can not set apn")
		}
	}

	if false {
		if bindex := MMGetBearer(); len(bindex) != 0 {
			if err := MMDeleteBearer(bindex); err != nil {
				log.Warnf("Can not delete old  beare %s", err.Error())
			}
		}

		if err := MMCreateBearer(apn, username, password); err != nil {
			log.Warnf("Can not add new bearer %s", err.Error())
		}
		//	MMSimpleConnect(apn, username, password)
	}

	if pdps, err := MMGetPDP(); err == nil {
		log.Warnf("Apn new configured: %v", pdps)
	} else {
		log.Error("Can not get apn new configured ", pdps)
	}

	return nil
}

func SendAtCommand(dev, cmdp string, timeouts ...time.Duration) (retstr string, err error) {
	var s *serial.Port
	cmd := cmdp
	devat := GetTTyAt(dev)
	//	fmt.Println("convert tty", dev, "->", devat)
	timeout := time.Millisecond * 1000
	if len(timeouts) != 0 {
		timeout = timeouts[0]
	}
	//	serialConfig := &serial.Config{Name: devat, Baud: 38400, ReadTimeout: timeout}
	serialConfig := &serial.Config{Name: devat, Baud: baud, ReadTimeout: timeout}
	s, err = serial.OpenPort(serialConfig)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer s.Close()
	//AT+CGDCONT?
	n, err := s.Write([]byte(cmd))
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(timeouts) >= 2 {
		time.Sleep(timeouts[1])
	} else {
		time.Sleep(time.Millisecond * 100)
	}

	buf := make([]byte, 1024)
	n, err = s.Read(buf)
	if err != nil {
		fmt.Println(err)
		return
	}
	retstr = string(buf[:n])
	//<CR><LF> =>  beginning of the current line -> new line
	//"AT+CGMI<CR>"
	//<CR><LF>Nokia<CR><LF>
	//<CR><LF>OK<CR><LF>
	retstr = strings.Replace(retstr, "\r\n", "\n", -1)                  //new line respone at command are \r \n
	retstr = strings.Replace(retstr, "\n\n", "\n", -1)                  //remove empty line
	retstr = strings.TrimPrefix(strings.TrimSuffix(retstr, "\n"), "\n") //trim newline at start and end

	if retregex := sregexp.New(`((?s:.*))OK`).FindStringSubmatch(retstr); len(retregex) != 0 {
		return retregex[1], nil
	} else {
		fmt.Println("\nSendAtCommand error\n", "__\n", cmdp, "=>", retstr, "\n")
		return "", errors.New("AT respone error, respone: " + retstr)
	}
	return
}

func PDPdEL(dev string, indexs []int) (errindex []int) {
	for _, v := range indexs {
		if _, err := SendAtCommand(dev, fmt.Sprintf("AT+CGDCONT=%d\r", v), time.Millisecond*200, time.Millisecond*100); err != nil {
			//			errindex = append(errimmcli -m 2 --create-bearer="apn=<APN Address Here>,user=<User Name Here>,password=<Password Here>"ndex, v)
			//			return
		}
	}
	return
}

func InitModem(dev string) (retstr string, err error) {
	return SendAtCommand(dev, "AT+CFUN=1\r", time.Millisecond*300, time.Millisecond*100) //full functionality
}

func MMInitModem() (err error) {

	mmservicefile := "/lib/systemd/system/ModemManager.service"
	// lineExecStart := "ExecStart=/usr/sbin/ModemManager --filter-policy=strict --debug --log-level=DEBUG --log-file=/var/log/mm.log"
	lineExecStart := "ExecStart=/usr/sbin/ModemManager --filter-policy=strict --debug --log-level=DEBUG"
	pattern := "ExecStart=(.+)"
	if !gogrep.FileIsMatchLine(mmservicefile, lineExecStart, true) {
		sutils.FileUpdateOrAdd(mmservicefile, lineExecStart, lineExecStart, pattern, true)
		sexec.ExecCommand("systemctl", "daemon-reload")
		sexec.ExecCommand("systemctl", "restart", "ModemManager")
	}
	if !gosystem.AppIsActive("ModemManager") {
		sexec.ExecCommand("systemctl", "start", "ModemManager")
	}
	// MMPDPDelAll()
	// MMAte1()
	// gonmmm.MMSendAtCommand("AT+CFUN=1")
	return nil
}

func SetApn(dev, apn string) (retstr string, err error) {
	for i := 0; i < 2; i++ {
		if ret, err1 := SendAtCommand(dev, fmt.Sprintf(`AT+CGDCONT=%d,"IP","%s"`+"\r", i, apn), time.Millisecond*500, time.Millisecond*100); err != nil {
			err = err1
			retstr += ret
		}
	}
	return
}

func MMSetApn(apn string) (retstr string, err error) {
	contypeMap := map[int]string{0: "IP", 1: "IPV4V6", 2: "IPV6"}
	//	contypeMap = map[int]string{0: "IP"}

	for i, iptype := range contypeMap {
		atcmd := fmt.Sprintf(`AT+CGDCONT=%d,"%s","%s"`, i, iptype, apn)
		if _, err1 := gonmmm.MMSendAtCommand(atcmd, time.Minute*4); err != nil {
			err = err1
			retstr = fmt.Sprintf(`%s [%s] `, retstr, err.Error())
		}
	}
	return
}

func MMPDPdEL(indexs []int) (errindex []int) {
	for _, v := range indexs {
		if _, err := gonmmm.MMSendAtCommand(fmt.Sprintf("+CGDCONT=%d", v)); err != nil {
			errindex = append(errindex, v)
		}
	}
	return
}

func MMPDPDelAll() bool {
	flag := true
	if pdps, err := MMGetPDP(); err == nil {
		for k := range pdps {
			if _, err := gonmmm.MMSendAtCommand(fmt.Sprintf("+CGDCONT=%s", k)); err != nil {
				flag = false
			}
		}
	}
	return flag
}

func MMPDPIsConfigured(apn string) bool { //auto delete empty apn
	//	return false
	numconfig := 3
	cnt := 0
	if pdps, err := MMGetPDP(); err == nil && len(pdps) != 0 {
		for _, v := range pdps {
			if v[1] == apn {
				cnt++
				return true
				if cnt == numconfig {
					return true
				}
			}
		}
	}
	return false
}

func GetPDP(dev string) (pdps []string, err error) { //auto delete empty apn
	var tmpstring string
	pdps = []string{}
	if tmpstring, err = SendAtCommand(dev, "AT+CGDCONT?\r", time.Millisecond*500, time.Second); err != nil {
		return
	}

	//+CGDCONT: 0,"IP","soracom.io","",0,0,0,0,0,0
	//+CGDCONT: 1,"IPV4V6","soracom.io","",0,0,0,0,0,0
	//+CGDCONT: 2,"IP","","",0,0,0,0,0,0
	//+CGDCONT: 3,"IPV6","soracom.io","",0,0,0,0,0,0
	//+CGDCONT: 4,"IPV4V6","soracom.io","",0,0,0,0,0,0
	//
	//OK
	//		split := strings.Split(tmpstring, "\n+CGDCONT: ")
	tmpstring = sutils.StringTrimLeftRightNewlineSpace(tmpstring)
	split := strings.Split(tmpstring, "\n")
	//	for i, j := range split {
	//		fmt.Printf("\nsplit[%d]=%s\n", i, j)
	//	}
	for i := 0; i < len(split); i++ {
		pdp := strings.Split(split[i], ",")
		//		fmt.Println(split[i], len(pdp))
		//
		//		for i1, j1 := range pdp {
		//			fmt.Printf("\npdp[%d]=%s\n", i1, j1)
		//		}
		if len(pdp) >= 6 { //10
			if len(pdp[2]) != 0 {
				pdps = append(pdps, strings.Replace(pdp[2], `"`, "", -1))
			} else {
				if index := sregexp.New("[0-9]+").FindString(pdp[0]); len(index) != 0 {
					if id, err := strconv.Atoi(index); err == nil {
						PDPdEL(dev, []int{id})
					}
				}
			}
		}
	}
	return
}

func MMCreateBearer(apn, username, passowrd string) (err error) {
	//	mmcli -m 0 --create-bearer="ip-type=ipv4,apn=soracom.io,user=sora,password=sora"
	//	mmparas := fmt.Sprintf(`--create-bearer='ip-type=ipv4,apn=%s,user=%s,password=%s'`, apn, username, passowrd)
	mmparas := fmt.Sprintf(`--create-bearer='apn=%s,user=%s,password=%s'`, apn, username, passowrd)

	if _, err := gonmmm.MMRunCommand(mmparas, time.Second*20); err == nil {
		return nil
	} else {
		//		log.Errorf("Command error: %s", mmparas)
		return err
	}
}

func MMDeleteBearer(bindex string) (err error) {
	//	mmcli -m 2 --create-bearer="apn=<APN Address Here>,user=<User Name Here>,password=<Password Here>"
	mmparas := fmt.Sprintf(`--delete-bearer=%s`, bindex)
	if _, err := gonmmm.MMRunCommand(mmparas, time.Second*2); err == nil {
		return nil
	} else {
		return err
	}
}

func MMGetPDP() (pdps map[string][]string, err error) { //auto delete empty apn
	var tmpstring string
	pdps = map[string][]string{}

	tmpstring, err = gonmmm.MMSendAtCommand("+CGDCONT?")
	if err != nil {
		return pdps, err
	}
	//+CGDCONT: 0,"IP","soracom.io","",0,0,0,0,0,0
	//+CGDCONT: 1,"IPV4V6","soracom.io","",0,0,0,0,0,0
	//+CGDCONT: 2,"IP","","",0,0,0,0,0,0
	//+CGDCONT: 3,"IPV6","soracom.io","",0,0,0,0,0,0
	//+CGDCONT: 4,"IPV4V6","soracom.io","",0,0,0,0,0,0
	//
	//OK
	//		split := strings.Split(tmpstring, "\n+CGDCONT: ")
	tmpstring = sutils.StringTrimLeftRightNewlineSpace(tmpstring)
	//	split := strings.Split(tmpstring, "\n")
	//	for i, j := range split {
	//		fmt.Printf("\nsplit[%d]=%s\n", i, j)
	//	}
	for _, pdp := range sutils.String2lines(tmpstring) {
		if slidePdp := sregexp.New(`\+CGDCONT:\s+([0-9]+),"([^"]+)","([^"]+)"`).FindStringSubmatch(pdp); len(slidePdp) != 0 {
			pdps[slidePdp[1]] = []string{slidePdp[2], slidePdp[3]}
		}
	}
	return
}

func Ate1(dev string) (retstr string, err error) {
	return SendAtCommand(dev, "ATE1\r", time.Millisecond*100, time.Millisecond*20)
}

func MMAte1() (err error) {
	_, err = gonmmm.MMSendAtCommand("ATE1")
	return err
}

func MMAte0() (err error) {
	_, err = gonmmm.MMSendAtCommand("ATE0")
	return err
}

func Ate0(dev string) (retstr string, err error) {
	return SendAtCommand(dev, "ATE0\r", time.Millisecond*100, time.Millisecond*20)
}
func GetSimNumber(dev string) (retstr string, err error) {
	//	time.Sleep(time.Second)
	if retstr, err = SendAtCommand(dev, "AT+CNUM\r", time.Millisecond*1000); err != nil {
		return
	}
	//	fmt.Println("Getsimnumber respone", retstr)
	if retregex := sregexp.New(`,"(.+)",`).FindStringSubmatch(retstr); len(retregex) != 0 {
		return retregex[1], nil
	} else {
		return "", errors.New("Sim not plugged in or loose:" + retstr)
	}
}

func GetListNetwork(dev string) (retstr string, err error) {
	if retstr, err = SendAtCommand(dev, "AT+COPS=?\r", time.Second*20, time.Second*20); err != nil {
		return
	}
	//+COPS: (2,"JP DOCOMO","DOCOMO","44010",7),(3,"SoftBank","SoftBank","44020",2),,(0,1,2,3,4),(0,1,2)
	if retregex := sregexp.New(`\+COPS:\s+(.+)`).FindStringSubmatch(retstr); len(retregex) != 0 { //once line
		return retregex[1], nil
	} else {
		return "", errors.New("Sim not plugged in or loose")
	}
}

func MMGetListNetwork() (retstr map[string][]string, err error) {
	//Found 4 networks:
	//21404 - Yoigo (umts, available)
	//21407 - Movistar (umts, current)
	//21401 - vodafone ES (umts, forbidden)
	//21403 - Orange (umts, forbidden)
	retstr = make(map[string][]string)
	mmretstr := ""
	if mmretstr, err = gonmmm.MMRunCommand("--3gpp-scan", time.Minute*5); err == nil {
		for _, v := range sutils.String2lines(mmretstr) {
			//				fmt.Println(v)
			if ret := sregexp.New(`([0-9]+)\s+-\s+([^\s]+)\s+\(([^,]+),\s+([^\s,\)]+)`).FindStringSubmatch(v); len(ret) != 0 {
				retstr[ret[1]] = []string{ret[2], ret[3], ret[4]}
			}
		}
	} else {
		return
	}

	return

	//	retstr, _, err = gonmmm.MMSendAtCommand("+COPS=?")
	//	if err != nil {
	//		return retstr, err
	//	}
	//
	//	+COPS: (2,"JP DOCOMO","DOCOMO","44010",7),(3,"SoftBank","SoftBank","44020",2),,(0,1,2,3,4),(0,1,2)
	//	if retregex := sregexp.New(`\+COPS:\s+(.+)`).FindStringSubmatch(retstr); len(retregex) != 0 {
	//		return retregex[1], nil
	//	} else {
	//		return "", errors.New("Sim not plugged in or loose")
	//	}
}

func GetCurrentOperator(dev string) (retstr string, err error) {
	if retstr, err = SendAtCommand(dev, "AT+COPS?\r", time.Second*5, time.Second*2); err != nil {
		return
	}
	//+COPS: 0,0,"NTT DOCOMO",7
	if retregex := sregexp.New(`\+COPS:\s+(.+)`).FindStringSubmatch(retstr); len(retregex) != 0 { //onece line
		return retregex[1], nil
	} else {
		return "", errors.New("Sim not plugged in or loose")
	}
}

func MMGetCurrentOperator() (retstr string, err error) {
	if true {
		if oname := sregexp.New(`operator name:\s+(.+)$`).FindStringSubmatch(MMStatsGSM()); len(oname) == 0 {
			return "", errors.New("Sim not plugged in or loose")
		} else {
			return oname[1], nil
		}
	} else {
		retstr, err = gonmmm.MMSendAtCommand("+COPS?")
		if err != nil {
			return retstr, err
		}
		//+COPS: 0,0,"NTT DOCOMO",7
		if retregex := sregexp.New(`\+COPS:\s+(.+)`).FindStringSubmatch(retstr); len(retregex) != 0 { //onece line
			return retregex[1], nil
		} else {
			return "", errors.New("Sim not plugged in or loose")
		}
	}
}

func MMConfigOperator(mccmnc string) (retstr string, err error) {
	//44010 docomo
	if len(mccmnc) == 0 {
		mccmnc = "44010"
	}
	retstr, err = gonmmm.MMRunCommand("--3gpp-register-in-operator=" + mccmnc)
	if err != nil {
		return retstr, err
	}
	return
	//+COPS: 0,0,"NTT DOCOMO",7
}

//func GetListOperator(dev string) (retstr map[string]string, err error) {
//	cops := GetListNetwork(dev)
//	if retregex := sregexp.New(`\+COPS:\s+(.+)`).FindStringSubmatch(retstr); len(retregex) != 0 {
//		return retregex[1], nil
//	} else {
//		return "", errors.New("Sim not plugged in or loose")
//	}
//}

func GetEMEI(dev string) (retstr string, err error) {
	if retstr, err = SendAtCommand(dev, "AT+CGSN\r", time.Millisecond*100); err != nil {
		return
	}
	if retregex := sregexp.New(`[^\s]+$`).FindStringSubmatch(retstr); len(retregex) != 0 {
		return retregex[1], nil
	} else {
		return "", errors.New("Can not get emei")
	}
}

func MMGetEMEI() (retstr string, err error) {
	if emei := sregexp.New(`imei:\s+(.+)$`).FindStringSubmatch(MMStatsGSM()); len(emei) == 0 {
		return "", errors.New("Can not get EMEI value")
	} else {
		return emei[1], nil
	}
}

func GetNetworkSignalStrength(dev string) (retstr string, err error) {
	if retstr, err = SendAtCommand(dev, "AT+CSQ\r", time.Millisecond*100); err != nil {
		return
	}
	if retregex := sregexp.New(`[0-9]+`).FindStringSubmatch(retstr); len(retregex) != 0 {
		return retregex[1], nil
	} else {
		return "", errors.New("Can not get emei")
	}
}

func GmsIsPlug(vendor, produc string) bool {
	cmd := fmt.Sprintf(`
#set -euo pipefail
IFS=$'\n\t'
VENDOR="%s"
PRODUCT="%s"
{ [[ $VENDOR ]] || [[ $PRODUCT ]]; }  && {
   vp=$(lsusb | grep -m 1 -ie ${VENDOR}:${PRODUCT} | awk '{print $6}')
}
[[ $vp ]] || {
   vp=$(lsusb | grep -m 1 -ie 'Huawei' -e Modem -e Networkcard | awk '{print $6}')
}
[[ $vp ]] && {
   VENDOR=${vp%:*}
   PRODUCT=${vp#*:}
   exit 0
} || {
   echo "Cannot find USB"
   exit 1
}`, vendor, produc)
	if _, _, err := sexec.ExecCommandShell(cmd, time.Second*1); err != nil {
		return false
	}
	return true
}

func SwichModeNetwork(vendor, produc string) bool {
	cmd := fmt.Sprintf(`
#set -euo pipefail
IFS=$'\n\t'
VENDOR="%s"
PRODUCT="%s"
{ [[ $VENDOR ]] || [[ $PRODUCT ]]; }  && {
   vp=$(lsusb | grep -m 1 -ie ${VENDOR}:${PRODUCT} | awk '{print $6}')
}
[[ $vp ]] || {
   vp=$(lsusb | grep -m 1 -ie 'Huawei' -e 'Modem/Networkcard' | awk '{print $6}')
}
[[ $vp ]] && {
   VENDOR=${vp%:*}
   PRODUCT=${vp#*:}
} || {
   echo "Cannot find USB"
   exit 0
}

#udevadm control --reload-rules
lsusb | grep "$VENDOR:$PRODUCT" | grep -ie 'Mass Storage Mode' &&  {
	usb_modeswitch -p $PRODUCT -v $VENDOR -J;
	true
}
`, vendor, produc)
	if _, _, err := sexec.ExecCommandShell(cmd, time.Second*1); err != nil {
		fmt.Println("Can not restart USB LTE")
		return false
	}
	return true
}

func ResetGSMUsb(vendor, produc string) bool {
	cmd := fmt.Sprintf(`
#set -euo pipefail
IFS=$'\n\t'
VENDOR="%s"
PRODUCT="%s"
{ [[ $VENDOR ]] || [[ $PRODUCT ]]; }  && {
   vp=$(lsusb | grep -m 1 -ie ${VENDOR}:${PRODUCT} | awk '{print $6}')
}
[[ $vp ]] || {
   vp=$(lsusb | grep -m 1 -ie 'Huawei' -e 'Modem/Networkcard' | awk '{print $6}')
}
[[ $vp ]] && {
   VENDOR=${vp%:*}
   PRODUCT=${vp#*:}
} || {
   echo "Cannot find USB"
   exit 0
}

#udevadm control --reload-rules
lsusb | grep "$VENDOR:$PRODUCT" | grep -ie 'Mass Storage Mode' &&  {
	usb_modeswitch -p $PRODUCT -v $VENDOR -J;
	sleep 10
	true
} || {
    udevadm trigger --attr-match="idVendor=${VENDOR}" --attr-match="idProduct=${PRODUCT}"
	sleep 60
}

exit 0

#reset usb
for DIR in $(find /sys/bus/usb/devices/ -maxdepth 1 -type l); do
  if [[ -f $DIR/idVendor && -f $DIR/idProduct &&
        $(cat $DIR/idVendor) == $VENDOR && $(cat $DIR/idProduct) == $PRODUCT ]]; then
    echo 0 > $DIR/authorized
    sleep 0.5
    echo 1 > $DIR/authorized
  fi
done
sleep 5
#swich usb networkcard mode
#usb_modeswitch -p $PRODUCT -v $VENDOR -H
usb_modeswitch -p $PRODUCT -v $VENDOR -J`, vendor, produc)
	if _, _, err := sexec.ExecCommandShell(cmd, time.Second*1); err != nil {
		fmt.Println("Can not restart USB LTE")
		return false
	}
	return true
}

func MMResetGSM() bool {
	if _, err := gonmmm.MMRunCommand("-r"); err == nil {
		return false
	} else {
		return true
	}
}

func MMStatsGSM() string {
	cmd := `index=$(mmcli -L | grep -oPe 'org[^\s]+' | grep -Poe '[0-9]+$')
	[[ $index ]] && mmcli -m ${index}
	`
	if stdout, _, err := sexec.ExecCommandShell(cmd, time.Second*1); err != nil {
		fmt.Println("Can not stats USB LTE")
	} else {
		return string(stdout)
	}
	return ""
}

// "Network operator: (2,\"JP DOCOMO\",\"DOCOMO\",\"44010\",7),(3,\"SoftBank\",\"SoftBank\",\"44020\",2),,(0,1,2,3,4),(0,1,2)",
// NTT DOCOMO
var _soracom_operator_list = []string{"docomo", "kddi", "softbank"}

func GetGsmDevice() string {
	cmd := `index=$(mmcli -L | grep -oPe 'org[^\s]+' | grep -Poe '[0-9]+$')
	gsmDev=$([[ $index ]] && mmcli -m ${index} | grep 'primary port' | grep -m 1 -Poe '[^\s]+$')
	[[ $gsmDev ]] && echo -n "${gsmDev}" || exit 1
	#gsmDev=$([[ $index ]] && mmcli -m ${index} | grep 'primary port' | grep -m 1 -Poe '[^\s]+$' || { [[ -e "/dev/ttyUSB2" ]] && echo -n "ttyUSB2"; })
	#[[ $gsmDev ]] && echo -n "${gsmDev}" || { lsusb | grep -q -m 1 -ie 'Huawei' -e Modem -e Networkcard && echo -n "ttyUSB2"; }
	`
	if stdout, _, err := sexec.ExecCommandShell(cmd, time.Second*1); err != nil {
		ResetGSMUsb("", "")
		//		fmt.Println("Can not get GsmDevice")
	} else {
		return sutils.StringTrimLeftRightNewlineSpace(string(stdout))
	}

	return ""
}

func MMGetBearer() string {
	// return gonmmm.MMGetField(`modem.generic.bearers.value\[1\]`)
	cmd := `index=$(mmcli -L | grep -oPe 'org[^\s]+' | grep -Poe '[0-9]+$')
	[[ $index ]] &&  {
	   bearerindex=$(mmcli -m ${index} | grep -Ee '^\s*Bearer' | grep -Poe '[0-9]+$' | head -n 1)
	   [[ ${bearerindex} ]] &&  echo -n "${bearerindex}"
	}
	`
	if stdout, _, err := sexec.ExecCommandShell(cmd, time.Second*1); err != nil {
		fmt.Println("Can not get GsmDevice")
	} else {
		return sutils.StringTrimLeftRightNewlineSpace(string(stdout))
	}
	return ""
}

func MMGetSimNumber() string {
	//Numbers  |                  own: 02021977184
	if pnum := sregexp.New(`(?:Numbers.+ )(\+?[0-9]+)`).FindStringSubmatch(MMStatsGSM()); len(pnum) == 0 {
		return ""
	} else {
		return pnum[1]
	}
}

func MMGetNetworkSignalStrength() (retstr string) {
	if sigs := sregexp.New(`signal quality:\s+([0-9]+)`).FindStringSubmatch(MMStatsGSM()); len(sigs) == 0 {
		return ""
	} else {
		return sigs[1]
	}
}
