package main

type Config struct {
	Neo4JUrl string
}

func GetConfig() Config {
	return Config{
		Neo4JUrl: "http://neo4j:keys@localhost:7474/",
	}
}
