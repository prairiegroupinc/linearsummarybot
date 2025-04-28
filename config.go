package main

import (
	_ "embed"
	"encoding/json"
	"log"

	"github.com/andreyvit/jsonfix"
	"github.com/prairiegroupinc/linearsummarybot/yearmonth"
)

var StatesToSkip = map[string]struct{}{}

//go:embed config.json
var configJSON []byte

type AppConfig struct {
	StatesToSkip    []string                      `json:"states_to_skip"`
	TagsToBuckets   map[string]string             `json:"tags_to_buckets"`
	DefaultCapacity int                           `json:"default_capacity"`
	ByMonth         map[yearmonth.YM]*MonthConfig `json:"months"`
}

type MonthConfig struct {
	Capacity int            `json:"capacity"`
	Budget   map[string]int `json:"budget"`
}

var config AppConfig

func loadConfig() {
	err := json.Unmarshal(jsonfix.Bytes(configJSON), &config)
	if err != nil {
		log.Fatalf("Failed to parse config.json: %v", err)
	}

	log.Printf("config = %s", must(json.MarshalIndent(&config, "", "  ")))

	for _, state := range config.StatesToSkip {
		StatesToSkip[state] = struct{}{}
	}
}
