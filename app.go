// verenav

package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
	"os"
//    "strings"

    "cloud.google.com/go/firestore"
    "google.golang.org/appengine"

    "github.com/deckarep/golang-set"
    "github.com/gorilla/mux"
)

type RequestBody struct {
    Urls []string `json:"urls"`
}

type GetResponseBody struct {
    Urls []string `json:"urls"`
    Next uint32 `json:"next"`
}

type PostResponseBody struct {
    Message string `json:"message"`
    Urls []string `json:"urls"`
}

type Resource struct {
    LocPath string
}

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/api/resource", GetResourceHandler).Methods("GET")
    r.HandleFunc("/api/resource", PostResourceHandler).Methods("POST")
    http.Handle("/api/", r)
	appengine.Main()
}

func GetResourceHandler(w http.ResponseWriter, r *http.Request) {
//    vars := mux.Vars(r)
//    begin := vars["begin"]
    res := GetResponseBody {
        Urls: nil,
    }
    json, _ := json.Marshal(res)
    fmt.Fprint(w, string(json))
}

func PostResourceHandler(w http.ResponseWriter, r *http.Request) {
    var reqBody RequestBody
    var resBody PostResponseBody
    dec := json.NewDecoder(r.Body)
    err := dec.Decode(&reqBody)
    if err != nil {
        resBody = PostResponseBody {
            Message: fmt.Sprintf("Error: %v", err),
        }
    } else {
        reqSet := mapset.NewSet();
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
		allDocRef := col.Doc("all")
        allDocSnap, err := allDocRef.Get(ctx)
		if err != nil {
			log.Fatalf("Failed to get document \"all\": %v", err)
		}
		allDocSet := mapset.NewSet()
		for _, key := range allDocSnap.Data()["keys"].([]interface{}) {
			if keystring, ok := key.(string); ok {
				allDocSet.Add(keystring)
			}
		}

		// deltaDocSnap, err := col.Doc("delta").Get(ctx)
		// if err != nil {
		// 	log.Fatalf("Failed to get document \"delta\": %v", err)
		// }
		//
		// failDocSnap, err := col.Doc("failed").Get(ctx)
		// if err != nil {
		// 	log.Fatalf("Failed to get document \"failed\": %v", err)
		// }
		//
		// qDocSnap := col.Doc("queue")
		// if err != nil {
		// 	log.Fatalf("Failed to get document \"queue\": %v", err)
		// }


        // difSet := reqSet.Difference(docSet)
        // added := []string{}
        // if difSet.Cardinality() != 0 {
        //     for v := range difSet.Iterator().C {
        //         if newUrl, ok := v.(string); ok {
        //             exists := strings.Contains(newUrl, "stat100.ameba.jp")
        //             if exists == false {
        //                 continue
        //             }
        //             _, _, err = client.Collection("resources").Add(ctx, map[string]interface{}{
        //                 "url": newUrl,
        //             })
        //             if err != nil {
        //                 log.Fatalf("Failed to add a Firestore document: %v", err)
        //             }
        //             added = append(added, newUrl)
        //         }
        //     }
        // }
		// // ここまでTransaction

        resBody = PostResponseBody {
            Urls: nil,
        }
    }
    json, _ := json.Marshal(resBody)
    fmt.Fprint(w, string(json))
}

func mapsetFromDocKey(ctx context.Context, docname string) mapset.Set {
	col := client.Collection(os.Getenv("COLLECTION_NAME"))
	// ここからTransaction
	docRef := col.Doc(docname)
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		log.Fatalf("Failed to get document \"%v\": %v", docname, err)
	}
	docSet := mapset.NewSet()
	for _, key := range docSnap.Data()["keys"].([]interface{}) {
		if keystring, ok := key.(string); ok {
			docSet.Add(keystring)
		}
	}
	return docSet
}
