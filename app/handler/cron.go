package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"google.golang.org/appengine"
	"google.golang.org/api/iterator"

	"github.com/deckarep/golang-set"
)

func MinCronResource(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	cgcs, err := storage.NewClient(ctx)
    if err != nil {
        log.Fatalf("failed to create gcs client : %v", err)
    }
	defer cgcs.Close()
	bkt := cgcs.Bucket(os.Getenv("BUCKET_NAME"))

	cfs, err := firestore.NewClient(ctx, os.Getenv("PROJECT_NAME"))
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer cfs.Close()
	collection := cfs.Collection(os.Getenv("COLLECTION_NAME"))

	refs, err := collection.DocumentRefs(ctx).GetAll()
	if err != nil {
		log.Fatalf("Failed to get documents: %v", err)
	}
	// ここからTransaction
	err = cfs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		var deltaDoc *firestore.DocumentRef
		var failedDoc *firestore.DocumentRef
		var queueDoc *firestore.DocumentRef

		allDocSet := mapset.NewSet()
		for _, doc := range refs {
			switch doc.ID {
			case "delta":
				deltaDoc = doc
			case "failed":
				failedDoc = doc
			case "queue":
				queueDoc = doc
			default:
				snap, err := tx.Get(doc)
				if err != nil {
					return err
				}
				allDocSet = allDocSet.Union(CreateSetFromDocument(snap))
			}
		}

		deltaDocSnap, err := tx.Get(deltaDoc)
		if err != nil {
			return err
		}
		deltaDocSet := CreateSetFromDocument(deltaDocSnap)

		failedDocSnap, err := tx.Get(failedDoc)
		if err != nil {
			return err
		}
		failedDocSet := CreateSetFromDocument(failedDocSnap)

		queueDocSnap, err := tx.Get(queueDoc)
		if err != nil {
			return err
		}
		queueDocSet := CreateSetFromDocument(queueDocSnap)

		diffSet := queueDocSet.Difference(allDocSet).Difference(deltaDocSet).Difference(failedDocSet)
		for key := range diffSet.Iterator().C {
			if strings.Contains(key.(string), "stat100.ameba.jp") {
				res, err := http.Get(fmt.Sprintf("https://%v", key.(string)))
				if err != nil {
					failedDocSet.Add(key)
					continue
				}
				writer := bkt.Object(key.(string)).NewWriter(ctx)
				ct := res.Header.Get("Content-Type")
				if ct != "" {
					writer.ContentType = ct
				}
				_, err = io.Copy(writer, res.Body)
				if err != nil {
					log.Fatalf("failed to create object `%v`: %v", key, err)
				}
				err = writer.Close()
				if err != nil {
					log.Fatalf("failed to close object `%v`: %v", key, err)
				}
				deltaDocSet.Add(key)
			}
		}

		err = tx.Set(deltaDoc, map[string]interface{}{
			"keys": deltaDocSet.ToSlice(),
		})
		if err != nil {
			log.Fatalf("Failed to update document: %v", err)
		}
		err = tx.Set(failedDoc, map[string]interface{}{
			"keys": failedDocSet.ToSlice(),
		})
		if err != nil {
			log.Fatalf("Failed to update document: %v", err)
		}

		return tx.Set(queueDoc, map[string]interface{}{
			"keys": []string{},
		})
	})
	// ここまでTransaction
	if err != nil {
		log.Fatalf("Failed to update document: %v", err)
	}
}

func DayCronResource(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	cgcs, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to create gcs client : %v", err)
	}
	defer cgcs.Close()

	allSet := mapset.NewSet()
	it := cgcs.Bucket(os.Getenv("BUCKET_NAME")).Objects(ctx, nil)
	for {
		obj, err := it.Next()
		if err == iterator.Done {
	        break
	    }
		if err != nil {
			log.Fatalf("failed to get objects: %v", err)
		}
		allSet.Add(obj.Name)
	}

	cfs, err := firestore.NewClient(ctx, os.Getenv("PROJECT_NAME"))
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer cfs.Close()
	collection := cfs.Collection(os.Getenv("COLLECTION_NAME"))

	allSlice := allSet.ToSlice()
	const maxKeys = 1000
	for i := 0; i <= len(allSlice) / maxKeys; i++ {
		var tl int
		if i == len(allSlice) / maxKeys {
			tl = len(allSlice)
		} else {
			tl = (i + 1) * maxKeys
		}
		docname := fmt.Sprintf("all%d", i + 1)
		_, err = collection.Doc(fmt.Sprintf(docname)).Set(ctx, map[string]interface{}{
			"keys": allSlice[(i * maxKeys):tl],
		})
		if err != nil {
			log.Fatalf("failed to set document '%s' : %v", docname, err)
		}
	}

	_, err = collection.Doc("delta").Set(ctx, map[string]interface{}{
		"keys": []string{},
	})
	if err != nil {
		log.Fatalf("failed to set document 'delta' : %v", err)
	}
}
