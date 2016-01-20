package main

import (
	"time"
)

type Keyid string

type PublicKey struct {
	Creation   time.Time
	Expiration time.Time
	Keyid      Keyid
}

type UserID struct {
	PublicKey PublicKey
	Data      string
}

type Signature struct {
	Issuer     Keyid
	Creation   time.Time
	Expiration time.Time
}
