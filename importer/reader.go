package main

/*
 Reads in a gpg file and produces a stream of
 PublicKeys
*/

import (
	puck_gpg "github.com/hockeypuck/openpgp"
	"log"
	"os"
)

type KeyChan chan *puck_gpg.PrimaryKey

func ReadKeys(filename string, output KeyChan) {
	reader, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	/*
	 This starts the reader thread.
	*/
	ch := puck_gpg.ReadKeys(reader)
	for val := range ch {
		if val.Error != nil {
			panic(val.Error)
		}
		output <- val.PrimaryKey
	}
	close(output)
}

func DumpKeys(input KeyChan) {

	for key := range input {
		kid := key.KeyID()

		log.Printf("Got key ID %s fpr %s", kid, key.Fingerprint())

		for _, uid := range key.UserIDs {
			for _, sig := range uid.Signatures {
				log.Printf("Sig: type %s", sig.SigType)
				if sig.IssuerKeyID() == kid {
					continue
				}
				switch sig.SigType {
				case 0x10, 0x11, 0x12, 0x13:
					signerKID := sig.IssuerKeyID()
					signeeKID := key.KeyID()
					log.Printf("Got Signature by %s on %s", signerKID, signeeKID)
				}
			}
		}
	}
}
