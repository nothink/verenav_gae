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
	r.HandleFunc("/_cron/minutely", handler.MinCronResource).Methods("GET")
	r.HandleFunc("/_cron/daily", handler.DayCronResource).Methods("GET")
	http.Handle("/", r)
	appengine.Main()
}
