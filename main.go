package main

import (
	"github.com/rcrowley/go-metrics"
	"os"
	"github.com/alexcesaro/log/stdlog"
	"log"
)

var app App

func main() {
	var conf Config

	conf = GetConfig()
	app = App{
			Config: conf,
			Logger: stdlog.GetFromFlags(),
		}
	
	if len(os.Args) == 1 {
		log.Fatal("Not enough args, foo")
	}
	filename := os.Args[1]

	log.Print("Connecting to Neo4j...")
	connect(&app)

	initMetrics(&app)
	
	keych := make(KeyChan)
	
	go metrics.Log(metrics.DefaultRegistry, 10e9, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
	
	log.Print("Launching key reader...")
	go ReadKeys(filename, keych)
	LoadKeys(app, keych)
}

func initMetrics(app *App) {
	app.KeyCounter = metrics.NewMeter()
	metrics.Register("key", app.KeyCounter)

	app.SigCounter = metrics.NewMeter()
	metrics.Register("sig", app.SigCounter)
}
