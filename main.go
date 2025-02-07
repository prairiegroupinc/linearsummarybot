package main

import (
	"flag"
	"fmt"
	"log"
	// The only allowed non-stdlib import, as provided.
)

func main() {
	log.SetFlags(0)

	onceFlag := flag.Bool("once", false, "Run once on launch")
	httpAddr := flag.String("http", "", "Listen address for HTTP server, e.g. :8080")
	flag.Parse()

	if *onceFlag {
		rep, err := buildReport()
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		fmt.Println(rep)
		return
	}

	if *httpAddr != "" {
		startWeb(*httpAddr)
		return
	}

	flag.Usage()
}
