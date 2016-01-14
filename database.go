package main

import (
	"fmt"
	"github.com/boltdb/bolt"
	"log"
)

func updateDatabase(name string, id *string) {

	err := database.Update(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte("tumblr"))

		if b == nil {
			fmt.Println("sheeeeeeeeeeit")
		}

		// Set the value "bar" for the key "foo".
		if err := b.Put([]byte(name), []byte(*id)); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

}
