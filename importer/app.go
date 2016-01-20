package main

import (
	"github.com/alexcesaro/log"
	"github.com/jmcvetta/neoism"
	"github.com/rcrowley/go-metrics"
)

type Config struct {
	Neo4JUrl string
	WorkDir  string
}

type App struct {
	Config     *Config
	GraphDB    *neoism.Database
	KeyCounter metrics.Meter
	SigCounter metrics.Meter
	Logger     log.Logger
}
