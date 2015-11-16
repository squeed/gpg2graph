package main

import (
	"bytes"
	"net/mail"
	"strings"
	"regexp"
)

type EmailInfo struct {
	name    string
	comment string
	email   string
	domain  string
}

func removeComments(address string) (string, string) {
	/*
	 * The "encouraged" format for UIDs
	 * in gpg is:
	 *	Name Name (comment) <email@domain>

	 *
	 * This separates those comments
	 */

	inComment := false
	inEsc := false //next char escaped

	mailbuf := bytes.NewBuffer(make([]byte, 0, len(address)))
	commentbuf := bytes.NewBuffer(make([]byte, 0, 10))

	for i := 0; i < len(address); i++ {
		char := address[i]

		if char == '\\' {
			inEsc = true
			continue
		}
		if inComment {
			if char == ')' {
				if inEsc {
					commentbuf.WriteByte(char)
					inEsc = false
				} else {
					inComment = false
				}
				continue
			}

			if inEsc {
				commentbuf.WriteByte('\\')
				inEsc = false
			}

			commentbuf.WriteByte(char)
		} else {
			if char == '(' {
				if inEsc {
					mailbuf.WriteByte(char)
					inEsc = false
				} else {
					inComment = true
				}
				continue
			} else {
				if inEsc {
					mailbuf.WriteByte('\\')
					inEsc = false
				}
				mailbuf.WriteByte(char)
			}
		}
	}

	return mailbuf.String(), commentbuf.String()
}

var mailre = regexp.MustCompile(`([^<]+)<(.+)>`)

func parseUID(address string) *EmailInfo {
	var name, email, domain string

	rest, comment := removeComments(address)
	
	response := mailre.FindStringSubmatch(rest)

	if response != nil {
		name = strings.TrimSpace(response[1])
		email = strings.TrimSpace(response[2])
		temp := strings.SplitN(email, "@", 2)
		if len(temp) == 2 {
			domain = temp[1]
		}
	} else {
		addr, err := mail.ParseAddress(rest)
		if err == nil {
			name = addr.Name
			email = addr.Address
			temp := strings.SplitN(email, "@", 2)
			if len(temp) == 2 {
				domain = temp[1]
			}
		}
		//app.Logger.Infof("could not understand UID", address, rest)
	}
	return &EmailInfo{name, comment, email, domain}
}
