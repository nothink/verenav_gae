package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"google.golang.org/appengine"

	"github.com/nothink/verenav_gae/handler"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/api/resource", handler.GetApiResource).Methods("GET")
	r.HandleFunc("/api/resource", handler.PostApiResource).Methods("POST")
	http.Handle("/api/", r)
	appengine.Main()
}
