package main

import (
	"github.com/jmcvetta/neoism"
	"github.com/rcrowley/go-metrics"
	"github.com/alexcesaro/log"
	)

type GraphDBConn *neoism.Database

type App struct {
	Config  Config
	GraphDB GraphDBConn
	KeyCounter metrics.Meter
	SigCounter metrics.Meter
	Logger log.Logger
}
