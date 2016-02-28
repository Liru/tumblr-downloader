package main

import (
	"fmt"
	"github.com/boltdb/bolt"
	"log"
)

func setupDatabase(userBlogs []*blog) {
	db, err := bolt.Open("tumblr-update.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	database = db

	err = db.Update(func(tx *bolt.Tx) error {
		b, boltErr := tx.CreateBucketIfNotExists([]byte("tumblr"))
		if boltErr != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		for _, blog := range userBlogs {
			v := b.Get([]byte(blog.name))
			if len(v) != 0 {
				blog.lastPostID = string(v) // TODO: Messy, probably.
			}
		}

		return nil
	})

	if err != nil {
		log.Fatal("database: ", err)
	}
}

func updateDatabase(name string, id string) {

	err := database.Update(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte("tumblr"))

		if b == nil {
			fmt.Println("sheeeeeeeeeeit")
		}

		// Set the value "bar" for the key "foo".
		if err := b.Put([]byte(name), []byte(id)); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal("database: ", err)
	}

}
