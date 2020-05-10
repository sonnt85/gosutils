package simplessh

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"os"
)


// RFC 4254 7.1
type channelForwardMsg struct {
	Addr  string
	Rport uint32
}

// See RFC 4254, section 7.2
type forwardedTCPPayload struct {
	Addr       string
	Port       uint32
	OriginAddr string
	OriginPort uint32
}

// RFC 4254 7.2
type channelOpenDirectMsg struct {
	Raddr string
	Rport uint32
	Laddr string
	Lport uint32
}

// CreateKeyPairFiles is the equivalent of running 'ssh-keygen -t rsa"'
func CreateKeyPairFiles(publicKeyPath, privateKeyPath string) error {

	privateKeyFile, err := os.Create(privateKeyPath)
	if err != nil {
		return err
	}
	defer privateKeyFile.Close()

	publicKeyFile, err := os.Create(publicKeyPath)
	if err != nil {
		return err
	}
	defer publicKeyFile.Close()

	return CreateKeyPair(publicKeyFile, privateKeyFile)
}

// CreateKeyPair creates a new SSH Key Pair writing the formatted keys to the corresponding io.Writers
func CreateKeyPair(publicKey, privateKey io.Writer) (err error) {
	k, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return err
	}
	privatePEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	err = pem.Encode(privateKey, privatePEM)
	if err != nil {
		return err
	}
	public, err := ssh.NewPublicKey(&k.PublicKey)
	if err != nil {
		return err
	}
	_, err = publicKey.Write(ssh.MarshalAuthorizedKey(public))
	return err
}

// CreateKeyPair creates a new SSH Key Pair
func CreateKeyPairBytes() (publicKey, privateKey []byte) {
	publicKey = nil
	privateKey = nil
	k, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return
	}
	privatePEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	var privateKeyBuffer bytes.Buffer
	err = pem.Encode(&privateKeyBuffer, privatePEM)
	if err != nil {
		return nil, nil
	}
	privateKey = privateKeyBuffer.Bytes()

	public, err := ssh.NewPublicKey(&k.PublicKey)
	if err != nil {
		return nil, nil
	}
	publicKey = ssh.MarshalAuthorizedKey(public)
	return publicKey, privateKey
}

// LoadPrivateKey loads a file at the provided path and attempts to load it into an ssh.Signer that can be used for SSH servers
func LoadPrivateKey(filePath string) (ssh.Signer, error) {

	privateBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(privateBytes)
}

//func KeyAuthCallback(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
//	user, err := auth.AuthenticateUser(conn.User(), key)
//	if err != nil {
//		log.Println("Fail to authenticate", conn, ":", err)
//		return nil, errors.New("invalid authentication")
//	}
//	return &ssh.Permissions{Extensions: map[string]string{"user_id": user.Id}}, nil
//}
