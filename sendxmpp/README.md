# go-sendxmpp

## about

A little tool to send messages to an XMPP contact or MUC inspired by (but not as powerful as)
[sendxmpp](https://sendxmpp.hostname.sk/).

## requirements

* [go](https://golang.org/)

## installation

If you have *[GOPATH](https://github.com/golang/go/wiki/SettingGOPATH)*
set just run this commands:

```plain
$ go get salsa.debian.org/mdosch/go-sendxmpp
$ go install salsa.debian.org/mdosch/go-sendxmpp
```

You will find the binary in `$GOPATH/bin` or, if set, `$GOBIN`.

## usage

You can either pipe a programs output to `go-sendxmpp`, write in your terminal (put \^D in a new
line to finish) or send the input from a file (`-m` or `--message`).

The account data is expected at `~/.sendxmpprc` if no other configuration file location is specified with
`-f` or `--file`. The configuration file is expected to be in the following format:

```plain
username: <your_username>
jserver: <jabber_server>
port: <jabber_port>
password: <your_jabber_password>
```

If no configuration file is present or if the values should be overridden it is possible to define the
account details via command line options:

```plain
 Usage: go-sendxmpp [-cdintx] [-f value] [--help] [--http-upload value] [-j value] [-m value] [-p value] [-r value] [-u value] [parameters ...]
 -c, --chatroom     Send message to a chatroom.
 -d, --debug        Show debugging info.
 -f, --file=value   Set configuration file. (Default: ~/.sendxmpprc)
     --help         Show help.
     --http-upload=value
                    Send a file via http-upload.
 -i, --interactive  Interactive mode (for use with e.g. 'tail -f').
 -j, --jserver=value
                    XMPP server address.
 -m, --message=value
                    Set file including the message.
 -n, --no-tls-verify
                    Skip verification of TLS certificates (not recommended).
 -p, --password=value
                    Password for XMPP account.
 -r, --resource=value
                    Set resource. When sending to a chatroom this is used as
                    'alias'. (Default: go-sendxmpp) [go-sendxmpp]
 -t, --tls          Use TLS.
 -u, --username=value
                    Username for XMPP account.
 -x, --start-tls    Use StartTLS.
```

### examples

Send a message to two recipients using a configuration file.

```bash
cat message.txt | ./go-sendxmpp -f ./sendxmpp recipient1@example.com recipient2@example.com
```

Send a message to two recipients directly defining account credentials.

```bash
cat message.txt | ./go-sendxmpp -u bob@example.com -j example.com -p swordfish recipient1@example.com recipient2@example.com
```

Send a message to two groupchats (`-c`) using a configuration file.

```bash
cat message.txt | ./go-sendxmpp -cf ./sendxmpp chat1@conference.example.com chat2@conference.example.com
```

Send file changes to two groupchats (`-c`) using a configuration file.

```bash
tail -f example.log | ./go-sendxmpp -cif ./sendxmpp chat1@conference.example.com chat2@conference.example.com
```
