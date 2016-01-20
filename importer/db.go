package main

import (
	puck_gpg "github.com/hockeypuck/openpgp"
	"github.com/jmcvetta/neoism"
	"log"
)

func connect(app *App) {
	url := app.Config.Neo4JUrl
	var db *neoism.Database
	db, err := neoism.Connect(url)
	if err != nil {
		log.Fatal("Could not connect to DB", err)
	}
	app.GraphDB = db
}

func doQueries(conn *neoism.Database, queries []string) {
	for _, s := range queries {
		q := neoism.CypherQuery{
			Statement: s,
		}
		err := conn.Cypher(&q)
		if err != nil {
			panic(err)
		}
	}
}

func addConstraints(conn *neoism.Database) {

	_, err := conn.CreateUniqueConstraint("Key", "keyid")
	if err != nil {
		app.Logger.Info("Failed to create constraint, probably already exists")
	}

	_, err = conn.CreateUniqueConstraint("UserID", "uuid")
	if err != nil {
		app.Logger.Info("Failed to create constraint, probably already exists")
	}
}

func addIndexes(conn *neoism.Database) {

	_, err := conn.CreateIndex("Key", "domain")
	if err != nil {
		app.Logger.Info("Failed to create index, probably already exists")
	}

	_, err = conn.CreateIndex("Key", "email")
	if err != nil {
		app.Logger.Info("Failed to create index, probably already exists")
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

	app.Logger.Debugf("Got key ID %s fpr %s", kid, key.Fingerprint())

	InsertPubKey(conn, key)
	app.KeyCounter.Mark(1)

	for _, uid := range key.UserIDs {
		InsertUID(conn, key, uid)

		for _, sig := range uid.Signatures {
			if sig.IssuerKeyID() == kid {
				continue
			}
			switch sig.SigType {
			case 0x10, 0x11, 0x12, 0x13:
				InsertSignature(conn, key, uid, sig)
			}
		}
	}
}

func InsertPubKey(conn *neoism.Database, k *puck_gpg.PrimaryKey) {

	cq0 := neoism.CypherQuery{
		Statement: `
			MERGE (n:Key {keyid: {keyid}})
			ON CREATE SET
			n.fingerprint = {fingerprint}
			ON MATCH SET
			n.fingerprint = {fingerprint};`,
		Parameters: neoism.Props{
			"keyid":       k.KeyID(),
			"fingerprint": k.Fingerprint()}}

	err := conn.Cypher(&cq0)
	if err != nil {
		panic(err)
	}
}

func InsertUID(conn *neoism.Database, key *puck_gpg.PrimaryKey, uid *puck_gpg.UserID) {
	kid := key.KeyID()
	app.Logger.Debugf("Inserting UID %s of %s", uid.Keywords, kid)

	parsed := parseUID(uid.Keywords)

	cq0 := neoism.CypherQuery{
		Statement: `
			MATCH 
				(k:Key {keyid: {keyid}})
			MERGE k-[r:HasID]-(i:UserID {
						keyword: {keyword}, 
						uuid: {uuid},
						name: {name},
						comment: {comment},
						email: {email},
						domain: {domain}
						})`,
		Parameters: neoism.Props{
			"keyid":   key.KeyID(),
			"keyword": uid.Keywords,
			"uuid":    uid.UUID,
			"name":    parsed.name,
			"comment": parsed.comment,
			"email":   parsed.email,
			"domain":  parsed.domain,
		},
	}

	err := conn.Cypher(&cq0)
	if err != nil {
		panic(err)
	}
}

/*
 Insert a signature in to the database.
*/
func InsertSignature(conn *neoism.Database, pubkey *puck_gpg.PrimaryKey, uid *puck_gpg.UserID, sig *puck_gpg.Signature) {

	signerKID := sig.IssuerKeyID()
	signeeKID := pubkey.KeyID()

	app.Logger.Debugf("Got Signature by %s on %s", signerKID, signeeKID)

	// Stub out the signer key, in case it's not yet in the DB
	q_signer := neoism.CypherQuery{
		Statement:  `MERGE (n:Key {keyid: {kid}});`,
		Parameters: neoism.Props{"kid": signerKID},
	}
	err := conn.Cypher(&q_signer)
	if err != nil {
		log.Fatal(err)
	}

	//Add the signature record
	q_signature := neoism.CypherQuery{
		Statement: `
			MATCH 
				(m:Key {keyid: {signee}})-[ii:HasID]-(i:UserID {uuid: {uuid}}), 
				(n:Key {keyid: {signer}})
			MERGE n-[r:SIGNS]->i`,
		Parameters: neoism.Props{
			"uuid":   uid.UUID,
			"signee": signeeKID,
			"signer": signerKID,
		},
	}
	err = conn.Cypher(&q_signature)
	if err != nil {
		log.Fatal(err)
	}

	app.SigCounter.Mark(1)
}
