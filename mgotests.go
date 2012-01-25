package main

import (
	"encoding/base64"
	"fmt"
	"log"

	"launchpad.net/gobson/bson"
	"launchpad.net/mgo"
)

func main() {
	session, err := mgo.Mongo("localhost")
	if err != nil {
		log.Fatal("connecting: " + err.Error())
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	session.SetSyncTimeout(2e9)
	err = session.Ping()
	if err != nil {
		log.Fatal("pinging: " + err.Error())
	}

	c := session.DB("test").C("things")
	err = c.RemoveAll(nil)
	if err != nil {
		log.Fatal("remove: " + err.Error())
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
			log.Fatal("insert: " + err.Error())
		}
	}

	// update an existing
	_, err = c.Upsert(&bson.M{"three": &bson.M{"$exists":true}}, &bson.M{"eight":"8"})
	if err != nil {
		log.Fatal("update: " + err.Error())
	}
	// update a non existing
	_, err = c.Upsert(&bson.M{"foo": &bson.M{"$exists":true}}, &bson.M{"seven":"7"})
	if err != nil {
		log.Fatal("update: " + err.Error())
	}

	// remove
	err = c.Remove(&bson.M{"eight": &bson.M{"$exists":true}})
	if err != nil {
		log.Fatal("remove: " + err.Error())
	}

	// insert escaped dot
	illegal := "foo.bar"
	encoded := base64.StdEncoding.EncodeToString([]byte(illegal))
	err = c.Insert(&bson.M{encoded:"fooball"})
	if err != nil {
		log.Fatal("insert: " + err.Error())
	}

	// iter all
	result := bson.M{}
	it := c.Find(nil).Iter()
	for it.Next(&result) {
		fmt.Println("********")
		for k,v := range result {
			fmt.Println(k, v)
		}
		result = nil
	}
	if it.Err() != nil {
		log.Fatal(it.Err())
	}

	// get escaped one
	result = nil
	q := c.Find(&bson.M{encoded: &bson.M{"$exists": true}})	
	err = q.One(&result)
	if err != nil {
		log.Fatal("get: " + err.Error())
	}

	input2 := []string{
		"7",
		"8",
		"9",
		"10",
		"11",
	}

	// insert input
	for _,v := range input2 {
		encoded = base64.StdEncoding.EncodeToString([]byte("foobee"))
		err = c.Insert(&bson.M{encoded:v})
		if err != nil {
			log.Fatal("insert: " + err.Error())
		}
	}

	// iter foobees
	result = nil
	encoded = base64.StdEncoding.EncodeToString([]byte("foobee"))
	it = c.Find(&bson.M{encoded:&bson.M{"$exists": true}}).Iter()
	for it.Next(&result) {
		fmt.Println("****foobees****")
		for k,v := range result {
			fmt.Println(k, v)
		}
		result = nil
	}
	if it.Err() != nil {
		log.Fatal(it.Err())
	}

	// insert similar values
	err = c.Insert(&bson.M{"hello":"world"})
	if err != nil {
		log.Fatal("insert: " + err.Error())
	}
	err = c.Insert(&bson.M{"hello":"war"})
	if err != nil {
		log.Fatal("insert: " + err.Error())
	}
	// test regexp
	result = nil
	it = c.Find(&bson.M{"hello": &bson.M{"$regex": "^w"}}).Iter()
	for it.Next(&result) {
		fmt.Println("****hellos****")
		for k,v := range result {
			fmt.Println(k, v)
		}
		result = nil
	}
	if it.Err() != nil {
		log.Fatal(it.Err())
	}

}
