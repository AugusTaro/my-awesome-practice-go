package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("Hello World!")
	http.HandleFunc("/", helloHandler)
	http.ListenAndServe(":8080", nil)
}
func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}
