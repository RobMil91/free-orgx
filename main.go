package main

import (
	"html/template"
	"log"
	"net/http"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("internal/adapters/htmlx/hello.html"))
	tmpl.Execute(w, map[string]string{"Name": "HTMX + Go"})
}

func main() {
	http.HandleFunc("/hello", helloHandler)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
