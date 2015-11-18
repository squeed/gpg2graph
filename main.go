package main

import (
	"flag"
	"github.com/alexcesaro/log/stdlog"
	"github.com/rcrowley/go-metrics"
	"log"
	"os"
	"path/filepath"
)

var app App

func main() {
	config := new(Config)
	
	flag.StringVar(&config.Neo4JUrl, "db", "http://neo4j:keys@localhost:7474/", "Neo4J URL")
	flag.StringVar(&config.WorkDir, "workdir", "", "The work dir for temporary files")
	
	flag.Parse()
	
	app = App{
		Config: config,
		Logger: stdlog.GetFromFlags(),
	}
	
	if app.Config.Neo4JUrl == "" {
		log.Fatal("db must be provided")
	}
	
	if app.Config.WorkDir == "" {
		log.Fatal("Workdir must be provided")
	}
	st, err := os.Stat(app.Config.WorkDir); if err != nil {log.Fatal(err)}
	if !st.Mode().IsDir() {
		log.Fatal("Workdir is not a directory")
	}
	app.Config.WorkDir, err = filepath.Abs(app.Config.WorkDir); if err != nil {log.Fatal(err)}
	
	if flag.NArg() == 0 {
		log.Fatal("key file must be provided")
	}
	
	filename := flag.Arg(0)


	log.Print("Connecting to Neo4j...")
	connect(&app)

	initMetrics(&app)

	keych := make(KeyChan)

	go metrics.Log(metrics.DefaultRegistry, 10e9, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))

	log.Print("Launching key reader...")
	go ReadKeys(filename, keych)

	LoadKeysBulk(&app, keych)
	//LoadKeys(app, keych)

}

func initMetrics(app *App) {
	app.KeyCounter = metrics.NewMeter()
	metrics.Register("key", app.KeyCounter)

	app.SigCounter = metrics.NewMeter()
	metrics.Register("sig", app.SigCounter)
}
