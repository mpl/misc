package main

import (
	"fmt"
	"log"
	"launchpad.net/gobson/bson"
	"launchpad.net/mgo"
)

func main() {
	session, err := mgo.Mongo("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)
	c := session.DB("test").C("things")

	err = c.RemoveAll(nil)
	if err != nil {
		log.Fatal("remove: " + err.String())
	}
	input := map[string]string{
		"one": "1",
		"two": "2",
		"three": "3",
		"four": "4",
		"five": "5",
	}

	// insert input
	for k,v := range input {
		err = c.Insert(&bson.M{k:v})
		if err != nil {
			log.Fatal("insert: " + err.String())
		}
	}

	// update 
	result := bson.M{}
	err = c.Update(bson.M{"three": bson.M{"$exists":true}}, &bson.M{"seven":"7"})

	// iter all
	q := c.Find(nil).Iter()
	for q.Next(&result) {
		fmt.Println("********")
		for k,v := range result {
			fmt.Println(k, v)
		}
		result = nil
	}
	if q.Err() != nil {
		log.Fatal(q.Err())
	}

}
