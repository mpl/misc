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

//	key := "permanode"
	result := bson.M{}
//	var result []bson.M
	q := c.Find(nil).Iter()
/*
	err = q.All(&result)
	if err != nil {
		log.Fatal(err)
	}
	for k,v := range result {
		fmt.Println(k, v)
	}
*/

	for q.Next(&result) {
		for k,v := range result {
			fmt.Println(k, v)
		}
	}
	if q.Err() != nil {
		log.Fatal(q.Err())
	}
//	err = q.One(&result)
}
