package main

import (
	"Gee/gee"
	"fmt"
	"log"
	"net/http"
)

func main() {
	r := gee.New()
	r.GET("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintf(writer, "URL.PATH %s", request.URL.Path)
	})
	log.Fatal(http.ListenAndServe(":9999", nil))
}
