package main

import (
	"fmt"
	"log"

	"github.com/blang/semver"
	"github.com/boltdb/bolt"
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

		storedVersion := string(b.Get([]byte("_VERSION_")))
		v, err := semver.Parse(storedVersion)
		if err != nil {
			// Usually means 0.0.0, which means old database.
			log.Println(err)
		}

		checkVersion(v)

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
			log.Println(`Bucket "tumblr" doesn't exist in database. Something went wrong.`)
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

func updateDatabaseVersion() {
	err := database.Update(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte("tumblr"))

		if b == nil {
			log.Println(`Bucket "tumblr" doesn't exist in database. Something went wrong.`)
		}

		// Set the value "bar" for the key "foo".
		if err := b.Put([]byte(`_VERSION_`),
			[]byte(currentVersion.String())); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal("database: ", err)
	}
}

func checkVersion(v semver.Version) {
	fmt.Println("Current version is", currentVersion)
	if v.LT(currentVersion) {
		forceCheck = true
		log.Println("Checking entire tumblrblog due to new version.")
	}
}
