# Simple Note API

## Start up
run `go run main.go` 

## Endpoints

GET `localhost:8080/notes` -> gets all of the notes in the database

POST `localhost:808/note` -> creates a note, provide the following request body fields:
```
{
    "Title": "...", 
    "Body": "..."
}
```

GET `localhost:8080/note/{id}` -> get a note by id

DELETE `localhost:8080/note/{id}` -> delete a note 