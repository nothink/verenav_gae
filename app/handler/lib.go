package handler

import (
	"cloud.google.com/go/firestore"
	"github.com/deckarep/golang-set"
)

func CreateSetFromDocument(doc *firestore.DocumentSnapshot) mapset.Set {
	set := mapset.NewSet()
	for _, key := range doc.Data()["keys"].([]interface{}) {
		if keystring, ok := key.(string); ok {
			set.Add(keystring)
		}
	}
	return set
}
