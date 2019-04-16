package main

import (
	"log"
	"os"
	"sync"
)

type webhookPayload struct {
	Delta      map[string][]int    `json:"delta"`
	ListFolder map[string][]string `json:"list_folder"`
}

var (
	errlog  = log.New(os.Stderr, "ERR ", log.Ldate|log.Ltime|log.Lshortfile)
	infolog = log.New(os.Stdout, "INFO ", log.Ldate|log.Ltime|log.Lshortfile)

	dropboxToken = os.Getenv("DROPBOX_ACCESS_TOKEN")
)

func main() {
	// required env vars
	if len(dropboxToken) == 0 {
		log.Fatal("Failed to fetch DROPBOX_ACCESS_TOKEN from ENV")
	}

	outputFolder := os.Getenv("OUTPUT_FOLDER")
	if len(outputFolder) == 0 {
		log.Fatal("Failed to fetch OUTPUT_FOLDER from ENV")
	}

	in := startPipeline(
		watchFolder(30),                  // produces list of files in user's folder
		filterFileTypes(".torrent"),    // ignore irrelevant file types
		stopSeenEntries(),              // don't process the same entry twice
		downloadToFolder(outputFolder), // download to NAS storage
		markAsProcessed(),              // rename in dropbox so that it's not processed again (even after server restart)
	)
	defer close(in) // probably redundant, we don't care about closing the pipeline when main exits

	in <- "whatever" // trigger initial listFolder

	wg := &sync.WaitGroup{}
	wg.Add(1)
	wg.Wait() // wait forever
}
