package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
)

const db_filepath = "notes.db"

type Note struct {
	Id    string
	Title string
	Body  string
}

type NewNote struct {
	Title string
	Body  string
}

func main() {
	_, err := initializeDatabase()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	handleRequests()
}

func initializeDatabase() (*bolt.DB, error) {
	db, err := bolt.Open(db_filepath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("could not open db: %v", err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists([]byte("DB"))
		if err != nil {
			return fmt.Errorf("could not create DB bucket: %v", err)
		}
		_, err = root.CreateBucketIfNotExists([]byte("NOTES"))
		if err != nil {
			return fmt.Errorf("could not create NOTES bucket: %v", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not set up database: %v", err)
	}
	return db, nil
}

func handleRequests() {
	router := mux.NewRouter().StrictSlash(true)

	router.Handle("/notes", AddContext(http.HandlerFunc(getNotes))).Methods("GET")

	router.Handle("/note", AddContext(http.HandlerFunc(createNote))).Methods("POST")

	router.Handle("/note/{id}", AddContext(http.HandlerFunc(getNote))).Methods("GET")
	router.Handle("/note/{id}", AddContext(http.HandlerFunc(deleteNote))).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":8080", router))
}

func AddContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "db_connection", db_filepath)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getNotes(w http.ResponseWriter, r *http.Request) {
	db, _ := bolt.Open(r.Context().Value("db_connection").(string), 0600, nil)
	defer db.Close()

	notes := []*Note{}
	if err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("DB")).Bucket([]byte("NOTES"))
		if bucket == nil {
			return errors.New("can't access notes")
		}

		return bucket.ForEach(func(k, v []byte) error {
			note := &Note{}
			if err2 := json.Unmarshal(v, &note); err2 != nil {
				return errors.New("can't unmarshall note")
			}
			notes = append(notes, note)
			return nil
		})
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		json.NewEncoder(w).Encode(notes)
	}
}

func getNote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	db, _ := bolt.Open(r.Context().Value("db_connection").(string), 0600, nil)
	defer db.Close()

	note := &Note{}
	if err := db.View(func(tx *bolt.Tx) error {
		res := tx.Bucket([]byte("DB")).Bucket([]byte("NOTES")).Get([]byte(id))
		if res == nil {
			return errors.New("can't find note")
		}

		if err2 := json.Unmarshal(res, &note); err2 != nil {
			return errors.New("can't unmarshall note")
		}

		return nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		json.NewEncoder(w).Encode(note)
	}
}

func createNote(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var newNote NewNote
	if err1 := json.Unmarshal(reqBody, &newNote); err1 != nil {
		http.Error(w, err1.Error(), http.StatusInternalServerError)
		return
	}

	id := time.Now().Format("20060102150405")
	note := Note{
		Id:    id,
		Title: newNote.Title,
		Body:  newNote.Body,
	}

	noteBytes, err2 := json.Marshal(note)
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
		return
	}

	db, _ := bolt.Open(r.Context().Value("db_connection").(string), 0600, nil)
	defer db.Close()

	if err := db.Update(func(tx *bolt.Tx) error {
		err3 := tx.Bucket([]byte("DB")).Bucket([]byte("NOTES")).Put([]byte(id), noteBytes)
		if err3 != nil {
			return errors.New("can't create note")
		}
		return nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		json.NewEncoder(w).Encode(note)
	}
}

func deleteNote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	db, _ := bolt.Open(r.Context().Value("db_connection").(string), 0600, nil)
	defer db.Close()

	if err := db.Update(func(tx *bolt.Tx) error {
		err2 := tx.Bucket([]byte("DB")).Bucket([]byte("NOTES")).Delete([]byte(id))
		if err2 != nil {
			return errors.New("can't delete note")
		}

		return nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		json.NewEncoder(w).Encode("deleted note " + id)
	}
}
