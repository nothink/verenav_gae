package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	//    "sort"

	"cloud.google.com/go/firestore"
	"google.golang.org/appengine"

	"github.com/deckarep/golang-set"
)

type RequestBody struct {
	Urls []string `json:"urls"`
}

type GetResponseBody struct {
	Urls []string `json:"urls"`
	Next int      `json:"next"`
}

type PostResponseBody struct {
	Count int      `json:"count"`
	Urls  []string `json:"urls"`
}

type Resource struct {
	LocPath string
}

func GetApiResource(w http.ResponseWriter, r *http.Request) {
	//    vars := mux.Vars(r)
	//    begin := vars["begin"]
	res := GetResponseBody{
		Urls: nil,
	}
	json, _ := json.Marshal(res)
	fmt.Fprint(w, string(json))
}

func PostApiResource(w http.ResponseWriter, r *http.Request) {
	var reqBody RequestBody
	var resBody PostResponseBody
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&reqBody)
	if err != nil {
		resBody = PostResponseBody{
			Count: -1,
		}
	} else {
		// requestされたURLをmapset化
		reqSet := mapset.NewSet()
		for _, url := range reqBody.Urls {
			reqSet.Add(url)
		}

		ctx := appengine.NewContext(r)
		client, err := firestore.NewClient(ctx, os.Getenv("PROJECT_NAME"))
		if err != nil {
			log.Fatalf("Failed to create Firestore client: %v", err)
		}
		defer client.Close()

		col := client.Collection(os.Getenv("COLLECTION_NAME"))

		// ここからTransaction

		queueDoc := col.Doc("queue")
		queueDocSnap, err := queueDoc.Get(ctx)
		if err != nil {
			log.Fatalf("Failed to get document \"queue\": %v", err)
		}
		queueDocSet := CreateSetFromDocument(queueDocSnap)

		newQueueSet := queueDocSet.Union(reqSet)
		_, err = queueDoc.Set(ctx, map[string]interface{}{
			"keys": newQueueSet.ToSlice(),
		})
		if err != nil {
			log.Fatalf("Failed to set document \"queue\": %v", err)
		}
		diffQueueSet := reqSet.Difference(queueDocSet)

		// ここまでTransaction

		resBody = PostResponseBody{
			Count: diffQueueSet.Cardinality(),
			Urls:  nil,
		}
	}
	json, _ := json.Marshal(resBody)
	fmt.Fprint(w, string(json))
}

func CreateSetFromDocument(doc *firestore.DocumentSnapshot) mapset.Set {
	set := mapset.NewSet()
	for _, key := range doc.Data()["keys"].([]interface{}) {
		if keystring, ok := key.(string); ok {
			set.Add(keystring)
		}
	}
	return set
}
