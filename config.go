package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

var globalConf confModel
var roomSq uint64 = 1

type confModel struct {
	RecapSecure string `json:"recapSecure"`
	Host        string `json:"host"`
	ResDir      string `json:"resourcedir"`
	ResRef      string `json:"resref"`
	Announce    string `json:"announce"`
	RecapServer string `json:"recapServer"`
	Production  bool   `json:"production"`
}

func initConf() {
	confpath := flag.String("config", "./.config.json", "config file path")
	flag.Parse()
	confFile, err := os.Open(*confpath)
	if err != nil {
		log.Fatal(err)
	}
	json.NewDecoder(confFile).Decode(&globalConf)
	log.Println(globalConf)
}
