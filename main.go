package main

import "os"
import "log"

var app App

func main() {
	log.Print("Welcome to Keygraph")
	var conf Config

	conf = GetConfig()
	app = App{Config: conf}

	log.Print("Connecting to Neo4j...")
	connect(&app)

	if len(os.Args) == 1 {
		log.Fatal("Not enough args, foo")
	}
	filename := os.Args[1]

	keych := make(KeyChan)

	log.Print("Launching key reader...")
	go ReadKeys(filename, keych)
	LoadKeys(app, keych)
	//DumpKeys(keych)

}
