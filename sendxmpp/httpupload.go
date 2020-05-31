// Copyright 2020 Martin Dosch.
// Use of this source code is governed by the BSD-2-clause
// license that can be found in the LICENSE file.

package sendxmpp

import (
	"bytes"
	"crypto/rand"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mattn/go-xmpp" // BSD-3-Clause"
)

func getID() string {
	b := make([]byte, 12)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	id := fmt.Sprintf("%x-%x-%x", b[0:4], b[4:8], b[8:])
	return id
}

func HttpUpload(client *xmpp.Client, jserver string, filePath string) string {

	// Created with https://github.com/miku/zek
	type IQDiscoItemsType struct {
		XMLName xml.Name `xml:"query"`
		Text    string   `xml:",chardata"`
		Xmlns   string   `xml:"xmlns,attr"`
		Item    []struct {
			Text string `xml:",chardata"`
			Jid  string `xml:"jid,attr"`
		} `xml:"item"`
	}

	// Created with https://github.com/miku/zek
	type IQDiscoInfoType struct {
		XMLName  xml.Name `xml:"query"`
		Text     string   `xml:",chardata"`
		Xmlns    string   `xml:"xmlns,attr"`
		Identity struct {
			Text     string `xml:",chardata"`
			Type     string `xml:"type,attr"`
			Name     string `xml:"name,attr"`
			Category string `xml:"category,attr"`
		} `xml:"identity"`
		Feature []struct {
			Text string `xml:",chardata"`
			Var  string `xml:"var,attr"`
		} `xml:"feature"`
		X []struct {
			Text  string `xml:",chardata"`
			Type  string `xml:"type,attr"`
			Xmlns string `xml:"xmlns,attr"`
			Field []struct {
				Text  string `xml:",chardata"`
				Type  string `xml:"type,attr"`
				Var   string `xml:"var,attr"`
				Value string `xml:"value"`
			} `xml:"field"`
		} `xml:"x"`
	}

	// Created with https://github.com/miku/zek
	type IQHttpUploadSlot struct {
		XMLName xml.Name `xml:"slot"`
		Text    string   `xml:",chardata"`
		Xmlns   string   `xml:"xmlns,attr"`
		Get     struct {
			Text string `xml:",chardata"`
			URL  string `xml:"url,attr"`
		} `xml:"get"`
		Put struct {
			Text string `xml:",chardata"`
			URL  string `xml:"url,attr"`
		} `xml:"put"`
	}

	var iqDiscoItemsXML IQDiscoItemsType
	var iqDiscoInfoXML IQDiscoInfoType
	var iqHttpUploadSlotXML IQHttpUploadSlot
	var uploadComponent string
	var maxFileSize int64

	// Get file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Fatal(err)
	}
	fileSize := fileInfo.Size()

	// Open File
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Read file
	buffer := make([]byte, fileSize)
	_, err = f.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}

	// Get mime type
	mimeType := http.DetectContentType(buffer)

	// Get file name
	fileName := filepath.Base(filePath)

	// Query server for disco#items
	id := getID()
	c := make(chan xmpp.IQ)
	go getIQ(client, id, c)
	_, err = client.RawInformation(client.JID(), jserver, id,
		"get", "<query xmlns='http://jabber.org/protocol/disco#items'/>")
	if err != nil {
		log.Fatal(err)
	}
	iqContent := <-c
	close(c)
	if iqContent.Type != "result" {
		log.Fatal("Error while disco#items query.")
	}
	err = xml.Unmarshal(iqContent.Query, &iqDiscoItemsXML)
	if err != nil {
		log.Fatal(err)
	}

	// Check the services reported by disco#items for the http upload service
	for _, r := range iqDiscoItemsXML.Item {
		id = getID()
		c := make(chan xmpp.IQ)
		go getIQ(client, id, c)
		_, err = client.RawInformation(client.JID(), r.Jid, id, "get",
			"<query xmlns='http://jabber.org/protocol/disco#info'/>")
		if err != nil {
			log.Fatal(err)
		}
		iqDiscoInfo := <-c
		close(c)
		if iqDiscoInfo.Type != "result" {
			log.Fatal("Error while disco#info query.")
		}
		err = xml.Unmarshal(iqDiscoInfo.Query, &iqDiscoInfoXML)
		if err != nil {
			log.Fatal(err)
		}

		if iqDiscoInfoXML.Identity.Type == "file" && iqDiscoInfoXML.Identity.Category == "store" {
			uploadComponent = r.Jid
		}

	}
	if uploadComponent == "" {
		log.Fatal("No http upload component found.")
	}
	for _, r := range iqDiscoInfoXML.X {
		for i, t := range r.Field {
			if t.Var == "max-file-size" && r.Field[i-1].Value == "urn:xmpp:http:upload:0" {
				maxFileSize, err = strconv.ParseInt(t.Value, 10, 64)
				if err != nil {
					log.Fatal("Error while checking server maximum http upload file size.")
				}
			}
		}
	}
	// Check if the file size doesn't exceed the maximum file size of the http upload
	// component if a maximum file size is reported, if not just continue and hope for
	// the best.
	if maxFileSize != 0 {
		if fileSize > maxFileSize {
			log.Fatal("File size " + strconv.FormatInt(fileSize/1024/1024, 10) +
				" MB is larger than the maximum file size allowed (" +
				strconv.FormatInt(maxFileSize/1024/1024, 10) + " MB).")
		}
	}

	// Request http upload slot
	id = getID()
	c = make(chan xmpp.IQ)
	go getIQ(client, id, c)
	_, err = client.RawInformation(client.JID(), uploadComponent, id, "get",
		"<request xmlns='urn:xmpp:http:upload:0' filename='"+
			fileName+"' size='"+strconv.FormatInt(fileSize, 10)+
			"' content-type='"+mimeType+"' />")
	if err != nil {
		log.Fatal(err)
	}
	uploadSlot := <-c
	close(c)
	if uploadSlot.Type != "result" {
		log.Fatal("Error while requesting upload slot.")
	}
	err = xml.Unmarshal(uploadSlot.Query, &iqHttpUploadSlotXML)
	if err != nil {
		log.Fatal(err)
	}

	// Upload file
	httpClient := &http.Client{}
	req, err := http.NewRequest(http.MethodPut, iqHttpUploadSlotXML.Put.URL, bytes.NewBuffer(buffer))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", mimeType)
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	// Test for http status code "200 OK" or "201 Created"
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		log.Fatal("Http upload failed.")
	}

	// Return http link
	return iqHttpUploadSlotXML.Get.URL
}

func getIQ(client *xmpp.Client, id string, c chan xmpp.IQ) {
	for {
		msg, err := client.Recv()
		if err != nil {
			log.Fatal(err)
		}

		switch v := msg.(type) {
		case xmpp.IQ:
			if v.ID == id {
				c <- v
				return
			}
		}
	}
}
