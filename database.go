package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/blang/semver"
	"github.com/boltdb/bolt"
)

var database *bolt.DB

func setupDatabase(userBlogs []*User) {
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
				blog.lastPostID, _ = strconv.ParseInt(string(v), 10, 64) // TODO: Messy, probably.
				blog.updateHighestPost(blog.lastPostID)
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

func updateDatabase(name string, id int64) {

	err := database.Update(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte("tumblr"))

		if b == nil {
			log.Println(`Bucket "tumblr" doesn't exist in database. Something went wrong.`)
		}

		if err := b.Put([]byte(name), []byte(strconv.FormatInt(id, 10))); err != nil {
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
			[]byte(cfg.version.String())); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal("database: ", err)
	}
}

func checkVersion(v semver.Version) {
	fmt.Println("Current version is", cfg.version)
	if v.LT(cfg.version) {
		cfg.forceCheck = true
		log.Println("Checking entire tumblrblog due to new version.")
	}
}
