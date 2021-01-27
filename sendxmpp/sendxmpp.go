// Copyright 2018 - 2020 Martin Dosch.
// Use of this source code is governed by the BSD-2-clause
// license that can be found in the LICENSE file.

package sendxmpp

//package main

import (
	"bufio"
	"crypto/tls"
	"errors"
	//	"fmt"
	"io"
	//	log "github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-xmpp" // BSD-3-Clause
	//	. "github.com/sonnt85/gosutils/sendxmpp"
)

type configuration struct {
	username string
	jserver  string
	port     string
	password string
}

// Opens the config file and returns the specified values
// for username, server and port.
func parseConfig(configPath string) (configuration, error) {

	var (
		output configuration
		err    error
	)

	// Use ~/.sendxmpprc if no config path is specified.
	if configPath == "" {
		// Get systems user config path.
		osConfigDir := os.Getenv("$XDG_CONFIG_HOME")
		if osConfigDir != "" {
			configPath = osConfigDir + "/.sendxmpprc"
		} else {
			// Get the current user.
			curUser, err := user.Current()
			if err != nil {
				return output, err
			}
			// Get home directory.
			home := curUser.HomeDir
			if home == "" {
				return output, errors.New("no home directory found")
			}
			configPath = home + "/.sendxmpprc"
		}
	}

	// Check that config file is existing.
	info, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		return output, err
	}
	// Only check file permissions if we are not running on windows.
	if runtime.GOOS != "windows" {
		// Check for file permissions. Must be 600 or 400.
		perm := info.Mode().Perm()
		permissions := strconv.FormatInt(int64(perm), 8)
		if permissions != "600" && permissions != "400" {
			return output, errors.New("Wrong permissions for " + configPath + ": " +
				permissions + " instead of 400 or 600.")
		}
	}

	// Open config file.
	file, err := os.Open(configPath)
	if err != nil {
		return output, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	// Read config file per line.
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "#") {
			continue
		}

		row := strings.Split(scanner.Text(), " ")

		switch row[0] {
		case "username:":
			output.username = row[1]
		case "jserver:":
			output.jserver = row[1]
		case "password:":
			output.password = row[1]
		case "port:":
			output.port = row[1]
		default:
			if len(row) >= 2 {
				if strings.Contains(scanner.Text(), ";") {
					output.username = strings.Split(row[0], ";")[0]
					output.jserver = strings.Split(row[0], ";")[1]
					output.password = row[1]
				} else {
					output.username = strings.Split(row[0], ":")[0]
					output.jserver = strings.Split(row[0], "@")[1]
					output.password = row[1]
				}
			}
		}
	}
	file.Close()

	return output, err
}

func readMessage(messageFilePath string) (string, error) {
	var (
		output string
		err    error
	)

	// Check that message file is existing.
	_, err = os.Stat(messageFilePath)
	if os.IsNotExist(err) {
		return output, err
	}

	// Open message file.
	file, err := os.Open(messageFilePath)
	if err != nil {
		return output, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {

		if output == "" {
			output = scanner.Text()
		} else {
			output = output + "\n" + scanner.Text()
		}
	}

	if err := scanner.Err(); err != nil {
		if err != io.EOF {
			return "", err
		}
	}

	file.Close()

	return output, err
}

type XMPPClient struct {
	Chatroom    bool
	Debug       bool
	File        string
	HttpUpload  string
	Interactive bool
	MessageFile string
	Password    string
	Resource    string
	Server      string
	SkipVerify  bool
	StartTLS    bool
	TLS         bool
	User        string
	Recipients  []string
	Message     string
}

func (c *XMPPClient) Reset() { //default config
	c.Chatroom = false
	c.Debug = false
	c.File = ""
	c.HttpUpload = ""
	c.Interactive = false
	c.MessageFile = ""
	c.Password = "yeudoilen"
	c.Resource = "sendxmpp"
	c.Server = "talk.google.com"
	c.SkipVerify = false
	c.StartTLS = true
	c.TLS = false
	c.User = "bluemoutain85@gmail.com"
	c.Message = ""
	c.Resource = "sendxmpp"
}

//				sendxmppCmd.Flags().StringSliceVarP(&, "to", "T", []string{"thanhson.rf@gmail.com"}, "Recipients slide string")
//	sendxmppCmd.Flags().StringVarP(&, "http-upload", "", "", "Send a file via http-upload.")
//	sendxmppCmd.Flags().BoolVarP(&, "debug", "d", false, "Show debugging info.")
//	sendxmppCmd.Flags().StringVarP(&, "jserver", "j", xmppclient.Server, "XMPP server address [talk.google.com].")
//	sendxmppCmd.Flags().StringVarP(&, "username", "u", xmppclient.User, "Username for XMPP account [user@gmail.com].")
//	sendxmppCmd.Flags().StringVarP(&, "password", "p", xmppclient.Password, "Password for XMPP account.")
//	sendxmppCmd.Flags().BoolVarP(&, "chatroom", "c", false, "Send message to a chatroom.")
//	sendxmppCmd.Flags().BoolVarP(&, "tls", "t", xmppclient.TLS, "Use TLS.")
//	sendxmppCmd.Flags().BoolVarP(&, "start-tls", "x", xmppclient.StartTLS, "Use StartTLS.")
//	sendxmppCmd.Flags().StringVarP(&, "resource", "r", "sendxmpp", "Set resource. "+
//		"When sending to a chatroom this is used as 'alias'. (Default: sendxmpp)")
//	sendxmppCmd.Flags().StringVarP(&, "file", "f", "", "Set configuration file. (Default: ~/.sendxmpprc)")
//	sendxmppCmd.Flags().StringVarP(&, "message", "m", "", "Set file including the message.")
//	sendxmppCmd.Flags().BoolVarP(&, "interactive", "i", false, "Interactive mode (for use with e.g. 'tail -f').")
//	sendxmppCmd.Flags().BoolVarP(&, "no-tls-verify", "n", false, "Skip verification of TLS certificates (not recommended).")
func NewXMPP(user, server, password string) *XMPPClient {
	conf := &XMPPClient{}
	conf.Reset()
	if len(user) > 0 {
		conf.User = user
	}
	if len(server) > 0 {
		conf.Server = server
	}
	if len(password) > 0 {
		conf.Password = password
		//		println("==========================>password:", conf.Password)
	}
	return conf
}

func (conf *XMPPClient) SendMessage(message string, recipients []string) (err error) { //default config
	var (
		user, server, password string
	)

	if len(recipients) == 0 {
		if len(conf.Recipients) == 0 {
			return errors.New("No recipient specified.")
		} else {
			recipients = conf.Recipients
		}
	}

	if len(message) == 0 && len(conf.Message) != 0 {
		message = conf.Message
	}

	// Quit if unreasonable TLS setting is set.
	if conf.StartTLS && conf.TLS {
		return errors.New("Use either TLS or StartTLS.")
	}

	// Check that all recipient JIDs are valid.
	for i, recipient := range recipients {
		validatedJid, err := MarshalJID(recipient)
		if err != nil {
			return err
		}
		recipients[i] = validatedJid
	}

	if conf.Debug {
		log.Printf("%+v\n", conf)
	}
	// Read configuration file if user, server or password is not specified.
	if conf.User == "" || conf.Server == "" || conf.Password == "" {
		// Read configuration from file.
		config, err := parseConfig(conf.File)
		if err != nil {
			log.Printf("Error parsing ", conf.File, ": ", err)
			return err
		}
		// Set connection options according to config.
		user = config.username
		server = config.jserver
		password = config.password
		if config.port != "" {
			server = server + ":" + config.port
		}
	}

	// Overwrite user if specified via command line flag.
	if conf.User != "" {
		user = conf.User
	}

	// Overwrite server if specified via command line flag.
	if conf.Server != "" {
		server = conf.Server
	}

	// Overwrite password if specified via command line flag.
	if conf.Password != "" {
		password = conf.Password
	}

	if (conf.HttpUpload != "") && (conf.Interactive || (conf.MessageFile != "")) {
		if conf.Interactive {
			return errors.New("Interactive mode and http upload can't" +
				" be used at the same time.")
		}
		if conf.MessageFile != "" {
			return errors.New("You can't send a message while using" +
				" http upload.")
		}
	}

	// Use ALPN
	var tlsConfig tls.Config
	tlsConfig.ServerName = strings.Split(user, "@")[1]
	tlsConfig.NextProtos = append(tlsConfig.NextProtos, "xmpp-client")
	tlsConfig.InsecureSkipVerify = conf.SkipVerify

	// Set XMPP connection options.
	options := xmpp.Options{
		Host:      server,
		User:      user,
		Resource:  conf.Resource,
		Password:  password,
		NoTLS:     !conf.TLS,
		StartTLS:  conf.StartTLS,
		Debug:     conf.Debug,
		TLSConfig: &tlsConfig,
	}

	// Read message from file.
	if conf.MessageFile != "" {
		message, err = readMessage(conf.MessageFile)
		if err != nil {
			return err
		}
	}

	// Connect to server.
	client, err := options.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()
	if conf.HttpUpload != "" {
		message = HttpUpload(client, tlsConfig.ServerName,
			conf.HttpUpload)
	}

	// Skip reading message if '-i' or '--interactive' is set to work with e.g. 'tail -f'.
	if !conf.Interactive {
		if message == "" {

			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {

				if message == "" {
					message = scanner.Text()
				} else {
					message = message + "\n" + scanner.Text()
				}
			}

			if err := scanner.Err(); err != nil {
				if err != io.EOF {
					// Close connection and quit.
					return err
				}
			}
		}
	}

	// Send message to chatroom(s) if the flag is set.
	if conf.Chatroom {

		for _, recipient := range recipients {

			// Join the MUC.
			_, err := client.JoinMUCNoHistory(recipient, conf.Resource)
			if err != nil {
				// Try to nicely close connection,
				// even if there was an error joining.
				return err
			}
		}

		// Send in endless loop (for usage with e.g. "tail -f").
		if conf.Interactive {
			for {
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				message = scanner.Text()
				for _, recipient := range recipients {
					_, err = client.Send(xmpp.Chat{Remote: recipient,
						Type: "groupchat", Text: message})
					if err != nil {
						// Try to nicely close connection,
						// even if there was an error sending.
						return err
					}
				}
			}
		} else {
			// Send the message.
			for _, recipient := range recipients {
				if conf.HttpUpload != "" {
					_, err = client.Send(xmpp.Chat{Remote: recipient,
						Type: "groupchat", Ooburl: message, Text: message})
				} else {
					_, err = client.Send(xmpp.Chat{Remote: recipient,
						Type: "groupchat", Text: message})
				}
				if err != nil {
					// Try to nicely close connection,
					// even if there was an error sending.
					return err
				}
			}
		}

		for _, recipient := range recipients {
			// After sending the message, leave the Muc
			_, err = client.LeaveMUC(recipient)
			if err != nil {
				log.Println(err)
			}
		}
	} else {
		// If the chatroom flag is not set, send message to contact(s).

		// Send in endless loop (for usage with e.g. "tail -f").
		if conf.Interactive {
			for {
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				message = scanner.Text()
				for _, recipient := range recipients {
					_, err = client.Send(xmpp.Chat{Remote: recipient,
						Type: "chat", Text: message})
					if err != nil {
						// Try to nicely close connection,
						// even if there was an error sending.
						return err
					}
				}
			}
		} else {
			for _, recipient := range recipients {
				if conf.HttpUpload != "" {
					_, err = client.Send(xmpp.Chat{Remote: recipient, Type: "chat",
						Ooburl: message, Text: message})
				} else {
					_, err = client.Send(xmpp.Chat{Remote: recipient, Type: "chat",
						Text: message})
				}
				if err != nil {
					// Try to nicely close connection,
					// even if there was an error sending.
					return err
				}
			}
		}
	}

	// Wait for a short time as some messages are not delievered by the server
	// if the connection is closed immediately after sending a message.
	time.Sleep(100 * time.Millisecond)

	// Close XMPP connection
	return nil
}
