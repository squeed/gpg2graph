package main

import (
	puck_gpg "github.com/hockeypuck/openpgp"
	"github.com/jmcvetta/neoism"
	"log"
)

/*
 Send stuff to things
*/

type GraphDBConn *neoism.Database

func connect(app *App) {
	url := app.Config.Neo4JUrl
	var db GraphDBConn
	db, err := neoism.Connect(url)
	if err != nil {
		log.Fatal("Could not connect to DB", err)
	}
	app.GraphDB = db
}

func addConstraints(conn *neoism.Database) {
	q := neoism.CypherQuery{
		Statement: `
		CREATE CONSTRAINT ON(k:Key) ASSERT k.keyid IS UNIQUE;
		`,
	}
	err := conn.Cypher(&q)
	if err != nil {
		panic(err)
	}
}

func LoadKeys(app App, in chan *puck_gpg.PrimaryKey) {
	for key := range in {
		LoadKey(app, key)
	}
}

func LoadKey(app App, key *puck_gpg.PrimaryKey) {
	conn := app.GraphDB
	kid := key.KeyID()

	log.Printf("Got key ID %s fpr %s", kid, key.Fingerprint())

	InsertPubKey(conn, key)

	for _, uid := range key.UserIDs {
		for _, sig := range uid.Signatures {
			if sig.IssuerKeyID() == kid {
				continue
			}
			switch sig.SigType {
			case 0x10, 0x11, 0x12, 0x13:
				InsertSignature(conn, key, sig)
			}
		}
	}
}

func InsertPubKey(conn *neoism.Database, k *puck_gpg.PrimaryKey) {
	name := "Unknown"
	for _, uid := range k.UserIDs {
		name = uid.Keywords
		break
	}

	cq0 := neoism.CypherQuery{
		Statement: `
			MERGE (n:Key {keyid: {keyid}})
			ON CREATE SET
			n.name = {name},
			n.fingerprint = {fingerprint}
			ON MATCH SET
			n.name = {name},
			n.fingerprint = {fingerprint};`,
		Parameters: neoism.Props{
			"keyid":       k.KeyID(),
			"name":        name,
			"fingerprint": k.Fingerprint()}}
	
	err := conn.Cypher(&cq0)
	if err != nil {
		panic(err)
	}
}

/*
 This assumes that the signature is on the UID
*/
func InsertSignature(conn *neoism.Database, pubkey *puck_gpg.PrimaryKey, sig *puck_gpg.Signature) {

	signerKID := sig.IssuerKeyID()
	signeeKID := pubkey.KeyID()

	log.Printf("Got Signature by %s on %s", signerKID, signeeKID)

	// Stub out the signer key, in case it's not yet in the DB
	q_signer := neoism.CypherQuery{
		Statement:  `MERGE (n:Key {keyid: {kid}});`,
		Parameters: neoism.Props{"kid": signerKID},
	}
	err := conn.Cypher(&q_signer)
	if err != nil {
		panic(err)
	}

	//Add the signature record
	q_signature := neoism.CypherQuery{
		Statement: `
			MATCH (m:Key {keyid: {signer}}), (n:Key {keyid: {signee}})
			MERGE (m)-[r:Sign { type: {type}, creation: {creation}}]->(n)`,
		Parameters: neoism.Props{
			"signee":   signeeKID,
			"signer":   signerKID,
			"type":     sig.SigType,
			"creation": sig.Creation,
		},
	}
	err = conn.Cypher(&q_signature)
	if err != nil {
		log.Fatal(err)
	}
}
