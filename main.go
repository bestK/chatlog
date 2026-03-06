package main

import (
	"log"
	"os"

	"github.com/sjzar/chatlog/cmd/chatlog"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if chatlog.IsCLIInvocation(os.Args[1:]) {
		chatlog.Execute()
		return
	}
	if err := runGUI(); err != nil {
		log.Fatal(err)
	}
}
