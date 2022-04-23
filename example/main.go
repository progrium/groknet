package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/progrium/groknet"
)

func main() {

	l, err := groknet.Listen(groknet.Config{
		Subdomain: "groknet123",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	fmt.Println("URL:", l.URL)

	http.Serve(l, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(resp, "Hello, %s!\n", l.Account)
	}))

}
