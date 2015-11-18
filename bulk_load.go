package main

import (
	"encoding/csv"
	puck_gpg "github.com/hockeypuck/openpgp"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/jmcvetta/neoism"
)

type KeyFiles struct {
	Key_file, Stub_key_file, Uid_file, Sig_file         *os.File
	Key_writer, Stub_key_writer, Uid_writer, Sig_writer *csv.Writer
}

func GetWriters(dirname string) *KeyFiles {

	ret := new(KeyFiles)
	var err error

	if dirname == "" {
		dirname, err = ioutil.TempDir("", "gpg2graph")
		if err != nil {
			log.Fatal(err)
		}
	}

	ret.Key_file, err = os.OpenFile(dirname+"/key.csv", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal(err)
	}
	ret.Key_writer = csv.NewWriter(ret.Key_file)

	ret.Stub_key_file, err = os.OpenFile(dirname+"/stub_key.csv", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	ret.Stub_key_writer = csv.NewWriter(ret.Stub_key_file)

	ret.Uid_file, err = os.OpenFile(dirname+"/uid.csv", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	ret.Uid_writer = csv.NewWriter(ret.Uid_file)

	ret.Sig_file, err = os.OpenFile(dirname+"/sig.csv", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	ret.Sig_writer = csv.NewWriter(ret.Sig_file)
	return ret
}

func (f *KeyFiles) Flush() {
	f.Key_writer.Flush()
	f.Stub_key_writer.Flush()
	f.Uid_writer.Flush()
	f.Sig_writer.Flush()

	f.Key_file.Sync()
	f.Stub_key_file.Sync()
	f.Uid_file.Sync()
	f.Sig_file.Sync()
}

func LoadKeysBulk(app *App, in chan *puck_gpg.PrimaryKey) {
	writers := GetWriters(app.Config.WorkDir)

	app.Logger.Info("Dumping keys to workdir")
	WriteKeysBulk(writers, in)
	
	app.Logger.Info("Loading to Neo4J")
	InsertKeysBulk(app, writers)
}

func q(app *App, query *neoism.CypherQuery) {
	app.Logger.Debug(query.Statement)
	err := app.GraphDB.Cypher(query)
	if err != nil {
		panic(err)
	}

}
func InsertKeysBulk(app *App, writers *KeyFiles) {

	keyQuery := neoism.CypherQuery{
		Statement: `
			USING PERIODIC COMMIT
			LOAD CSV FROM {filename} AS line
			MERGE 
				(n:Key {keyid: line[0]})
			ON CREATE SET
				n.fingerprint = line[1]
			ON MATCH SET
				n.fingerprint = line[1];`,
		Parameters: neoism.Props{
			"filename": "file://" + writers.Key_file.Name(),
		}}

	q(app, &keyQuery)

	stubKeyQuery := neoism.CypherQuery{
		Statement: `
			USING PERIODIC COMMIT
			LOAD CSV FROM {filename} AS line
			MERGE (n:Key {keyid: line[0]})
			`,
		Parameters: neoism.Props{
			"filename": "file://" + writers.Stub_key_file.Name(),
		}}
	q(app, &stubKeyQuery)

	uidQuery := neoism.CypherQuery{
		Statement: `
			USING PERIODIC COMMIT
			LOAD CSV FROM {filename} AS line
			MATCH (k:Key {keyid : line[0]})
			MERGE (i:UserID { uuid: line[2]})
			FOREACH(ignoreMe IN CASE WHEN trim(line[1]) <> "" THEN [1] ELSE [] END | SET i.keyword = line[1])
			FOREACH(ignoreMe IN CASE WHEN trim(line[3]) <> "" THEN [1] ELSE [] END | SET i.name = line[3])
			FOREACH(ignoreMe IN CASE WHEN trim(line[4]) <> "" THEN [1] ELSE [] END | SET i.comment = line[4])
			FOREACH(ignoreMe IN CASE WHEN trim(line[5]) <> "" THEN [1] ELSE [] END | SET i.email = line[5])
			FOREACH(ignoreMe IN CASE WHEN trim(line[6]) <> "" THEN [1] ELSE [] END | SET i.domain = line[6])
			MERGE k-[r:HasID]-i;
		`,
		Parameters: neoism.Props{
			"filename": "file://" + writers.Uid_file.Name(),
		}}

	q(app, &uidQuery)

	sigQuery := neoism.CypherQuery{
		Statement: `
			USING PERIODIC COMMIT
			LOAD CSV FROM {filename} AS line
			MATCH 
				(to:Key {keyid: line[0]})-[ii:HasID]-(to_id:UserID {uuid: line[2]}),
				(from:Key {keyid: line[1]})
			MERGE
				from-[r:SIGNS]->(to_id)
		`,
		Parameters: neoism.Props{
			"filename": "file://" + writers.Sig_file.Name(),
		}}
	q(app, &sigQuery)
}

func WriteKeysBulk(writers *KeyFiles, in chan *puck_gpg.PrimaryKey) {

	for key := range in {
		keyID := key.KeyID()
		WriteKey(writers.Key_writer, key)
		app.KeyCounter.Mark(1)
		for _, uid := range key.UserIDs {
			WriteUID(writers.Uid_writer, key, uid)

			for _, sig := range uid.Signatures {
				if sig.IssuerKeyID() == keyID {
					continue
				}
				switch sig.SigType {
				case 0x10, 0x11, 0x12, 0x13:
					//InsertSignature(conn, key, uid, sig)
					WriteSignature(writers.Stub_key_writer, writers.Sig_writer, key, uid, sig)
					app.SigCounter.Mark(1)
				}
			}
		}
	}
	writers.Flush()
}

func WriteKey(key_writer *csv.Writer, key *puck_gpg.PrimaryKey) {
	record := []string{key.KeyID(), key.Fingerprint(), strconv.FormatInt(key.Creation.Unix(), 10)}
	err := key_writer.Write(record)
	if err != nil {
		log.Fatal(err)
	}
}

func WriteUID(uid_writer *csv.Writer, key *puck_gpg.PrimaryKey, uid *puck_gpg.UserID) {
	parsed := parseUID(uid.Keywords)

	record := []string{
		key.KeyID(),
		uid.Keywords,
		uid.UUID,
		parsed.name,
		parsed.comment,
		parsed.email,
		parsed.domain,
	}
	err := uid_writer.Write(record)
	if err != nil {
		log.Fatal(err)
	}
}

func WriteSignature(stub_key_writer *csv.Writer, sig_writer *csv.Writer, pubkey *puck_gpg.PrimaryKey, uid *puck_gpg.UserID, sig *puck_gpg.Signature) {
	signerKID := sig.IssuerKeyID()
	signeeKID := pubkey.KeyID()

	stubRecord := []string{signerKID}
	err := stub_key_writer.Write(stubRecord)
	if err != nil {
		log.Fatal(err)
	}

	sigRecord := []string{signeeKID, signerKID, uid.UUID}
	err = sig_writer.Write(sigRecord)
	if err != nil {
		log.Fatal(err)
	}
}
