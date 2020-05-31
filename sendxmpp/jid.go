// Copyright 2014 The Mellium Contributors.
// Use of this source code is governed by the BSD-2-clause
// license that can be found in the LICENSE-mellium file.

// Original taken from mellium.im/xmpp/jid (BSD-2-Clause) and adjusted for my needs.
// Copyright 2020 Martin Dosch

package sendxmpp

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// MarshalJID checks that JIDs include localpart and serverpart
// and return it marshalled. Shamelessly stolen from
// mellium.im/xmpp/jid
func MarshalJID(input string) (string, error) {

	var (
		err          error
		localpart    string
		domainpart   string
		resourcepart string
	)

	s := input

	// Remove any portion from the first '/' character to the end of the
	// string (if there is a '/' character present).
	sep := strings.Index(s, "/")

	if sep == -1 {
		resourcepart = ""
	} else {
		// If the resource part exists, make sure it isn't empty.
		if sep == len(s)-1 {
			return input, errors.New("Invalid JID" + input + ": The resourcepart must be larger than 0 bytes")
		}
		resourcepart = s[sep+1:]
		s = s[:sep]
	}

	// Remove any portion from the beginning of the string to the first
	// '@' character (if there is an '@' character present).

	sep = strings.Index(s, "@")

	switch {
	case sep == -1:
		// There is no @ sign, and therefore no localpart.
		return input, errors.New("Invalid JID: " + input)
	case sep == 0:
		// The JID starts with an @ sign (invalid empty localpart)
		err = errors.New("Invalid JID:" + input)
		return input, err
	default:
		domainpart = s[sep+1:]
		localpart = s[:sep]
	}

	// We'll throw out any trailing dots on domainparts, since they're ignored:
	//
	//    If the domainpart includes a final character considered to be a label
	//    separator (dot) by [RFC1034], this character MUST be stripped from
	//    the domainpart before the JID of which it is a part is used for the
	//    purpose of routing an XML stanza, comparing against another JID, or
	//    constructing an XMPP URI or IRI [RFC5122].  In particular, such a
	//    character MUST be stripped before any other canonicalization steps
	//    are taken.

	domainpart = strings.TrimSuffix(domainpart, ".")

	if !utf8.ValidString(localpart) || !utf8.ValidString(domainpart) || !utf8.ValidString(resourcepart) {
		return input, errors.New("Invalid JID: " + input)
	}

	if localpart == "" || domainpart == "" {
		return input, errors.New("Invalid JID: " + input)
	}

	if resourcepart == "" {
		return localpart + "@" + domainpart, err
	}
	return localpart + "@" + domainpart + "/" + resourcepart, err
}
