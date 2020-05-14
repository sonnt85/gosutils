package sshclient_test

import (
	"fmt"
	"github.com/sonnt85/gosutils/sshclient"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"testing"
)

var password = `-----BEGIN RSA PRIVATE KEY-----
MIIEpgIBAAKCAQEAsJeD1Y1It9S7eO3S3PpVT0ZN88zeej3eCaHLiY8h2H664KPb
tnWhnkdijYYC30SgLGhPHIrMzLSsFWE0tncvqmbV0FMkUv5mJB22KxxG+MFsLQhC
hHNobgZQpz3xo3Y/HvBk6NWACh2mnA89D0Xh9yP33gonLAMdnEUF1gXgaB+iqsnc
naEAslDwR/tLnidvFzn5PF1LymYAETNJ0nDcdzF8CCt34sHOlQmaq/9h9JOh1TOu
xCjjULuMWp0nNAPoyA4LpAdlrxMJbgFLNUXqgqdnoZHQRI5XrdClyv6qm6kspuyx
j7mYG9v6n6Nwk22hIlfFuqWjXMIJyTFAzG8YWQIDAQABAoIBAQCiIrr8e7fkcQGf
ylvsYDuriZVQ3yz1d5BBr7e9GRmuOM1EK64zHFXDiS9HWV+RtuSJYUwhnJ7k5I2L
I7DORygQgFKX735OZR1K06zKcDAJfS3hOtA34+5h9pJeu1T9DDhwI6/CxyPEJe0v
JB6fwz3xN6kAyLmmg0XQkN8G3mZnsf1FjbuUWSuYkfQeCtzX7IU9g0mjfiqUtcje
kfwgmbGFuj7iabwWA+e2UiEAeKG9MSjg8kXcZZApbDkfjJwqmmPwtwb5r6n4Rydd
pCMCPD9o1SXeyljGRHvd2aMEDIfpkVZJAHryPX5/dY/gX25U9gNDV6YTiHX3lqxe
O4pOak3BAoGBAN6BzaBsPHfpk+C5T24n2WX4Zd+lC6MaihaEMGNqYTz1aieXioad
yPRTpUg7l7+njpZovC9fh1JVltxOnRtFuxl22EynCX67MfjwMpeGwSvuV5zjhIoA
eRdw+dqat6lQFmacjiWxqmFsvRFs3U40xuVbzYJSa0XqL57QWsommtHdAoGBAMss
ZKKQTFol0dqL75EkX/wKwx6mjQIr+AlA7eNzjq/C8Os/52LHnk9hSuZzFjGrM4uy
Dnfk1EtUxl0epPfWBebW+ZJ3aD+U/+ta44vIHdjrHhaXYuSuy/rYh3yXw9kxMLeZ
caQBFJIzwgNdFVdLLm7iC62i4Lm34g/Nqezljv6tAoGBAIuGyfK27JQlHF3m1jA1
PNX8laVQUaPNmJnV+qHcq20WV6LMHEmd182eRh6tf9LmtzsKIjdyp+CxWxB7G3lm
mJS3OZuXgxS9PfDkblUmYyuxIa933DzNXyGb7pFuQ40gc2uU8G4iorzE+ypaIcxQ
vAhHMO9vz2TgHUxxSv1Ih/zhAoGBALx5xjF4IxxNkUtoHSlL0S8C3NcGMjEdkM8k
yIoDnQ43jT7u3TupapbA7raxdJlG9F5XI0zdnoLzdcDUuLygcoEeVA8nbjHtiytN
+WCml+mu0w6qCTeTX+6oB6fxMeG93C+1zNITnn2yPfzY0P9V4xFB6Qt+2XHvv2ph
o4z7t5dRAoGBAJ9M1SkT7/0NazGmfV1qqkuJATAGMaqKRpIYmI1mTPaLATG1WU99
5v4uxdGlSE0/fQiDNFUwYn7oQpQ60sB7jXRaW564KEUF/z4QFaOnwPT6cYNoQP5V
0Rs9NUU0KfJbejL+YA/irToklJcRsuPi5DAlAaDtnd9kkK+aXmtRgpgR
-----END RSA PRIVATE KEY-----`

func TestClientConfig() {
	fmt.Println("")
	config := &ssh.ClientConfig{
		User: "public",
		Auth: []ssh.AuthMethod{
			ssh.Password("licpub@321"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := sshclient.Dial("tcp", "www.ehomevn.com:443", config)
	if err != nil {
		log.Fatalf("unable to connect: %s", err)
	}
	defer conn.Close()

	laddr, _ := net.ResolveTCPAddr("tcp", "localhost:65534")
	raddr, _ := net.ResolveTCPAddr("tcp", "localhost:2222")
	err = conn.LocalForward(nil, laddr, raddr)
	if err != nil {
		log.Fatalf("unable to forward local port: %s", err)
	}

}

func TestFull(t *testing.T) {
	var sclient *sshclient.Client
	sclient = sshclient.NewClient("public", "www.ehomevn.com:443", password)
	if sclient.Dial() != nil {
		t.Errorf("Cannot connect to server")
		return
	} else {
		t.Errorf("connected to server")
	}

	stdout, stderr, err := sclient.Run(`bash -c "sleep 1; echo done"`)
	if err != nil {
		t.Errorf("Error: %s", err)
	}

	t.Logf("%s\n%s\n", stdout, stderr)
	t.Logf("Start interactive bash")
	sclient.Shell("bash")
	t.Log("Done bash")
	sclient.Wait()
}
