package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gobuffalo/packr"
)

const port = 8080

var box packr.Box

func handler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	io.WriteString(w, box.String(path))
}

func main() {
	port := flag.Int("port", 8080, "The port that the HTTP server listens on")
	ip := flag.String("ip", "127.0.0.1", "The ip that the HTTP server listens on")
	flag.Parse()

	box = packr.NewBox("./static")

	fmt.Println("Welcome to modpack-editor!")
	fmt.Println("Press CTRL+C to exit.")
	fmt.Printf("Listening on port %d, accessible at http://127.0.0.1:%d/", *port, *port)

	http.HandleFunc("/", handler)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", *ip, *port), nil)
	if err != nil {
		log.Println("Error starting server:")
		log.Fatal(err)
	}
}
