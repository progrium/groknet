package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/progrium/groknet"
)

func main() {
	log.Println("connecting...")
	l, err := groknet.Listen(groknet.Config{})
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	log.Println("URL:", l.URL)

	log.Println("tunneling...")
	http.Serve(l, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(resp, "Hello, %s!\n", l.Account)
	}))

}
