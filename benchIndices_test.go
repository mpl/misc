package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

var cdms = "cdms.db"

func hitCDMS() {
	db, err := sql.Open("sqlite3", cdms)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec("select count(*) from line where origin = 'jpl' and species = 'CO' and frequency < 120000")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("delete from line where origin = 'cdms' and species = 'HCNH+' and frequency > 110000")
	if err != nil {
		log.Fatal(err)
	}
}

func BenchmarkCachedCdms(b *testing.B) {
	for i := 0; i < b.N; i++ {
		hitCDMS()
	}
}

var treesdb = "trees.db"

func init() {
//	createTrees(false, true)
}

func createTrees(indexed bool, overwrite bool) {
	if _, err := os.Stat(treesdb); err == nil {
		if overwrite {
			err := os.Remove(treesdb)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal("treesdb already exists and overwrite not set")
		}
	}
	db, err := sql.Open("sqlite3", treesdb)
	if err != nil {
		log.Fatal("open: " + err.Error())
	}
	defer db.Close()
	query := "create table trees (" +
		"'species' text, 'country' text, 'leavy' bool, 'foliagecolor' text," +
		"'trunkcolor' text, 'height' int, 'width' int, 'count' int, 'sick' bool)"
//		"'trunkcolor' text, 'height' int, 'width' int, 'count' int, 'sick' bool," +
//		" constraint loose unique (species, country, leavy, foliagecolor, trunkcolor, height, width, sick))"
	_, err = db.Exec(query)
	if err != nil {
		log.Fatal("create: " + err.Error())
	}
	if indexed {
		_, err = db.Exec("create index leavyspeciescountry on trees ('leavy', 'species', 'country')")
		if err != nil {
			log.Fatal("create index: " + err.Error())
		}
	}

	// populate the DB
	allSpecies := []string{"chêne", "bouleau", "peuplier", "hêtre", "charmille", "noyer", "sapin", "if", "pin", "mélèze", "redwood", "sequoia"}
	allCountries := []string{"fr", "de", "it", "es", "no", "fi", "se", "uk", "si", "sk", "au"}
	allColors := []string{"brown", "red", "grey", "green", "yellow", "orange"}

	j := 0
	for j < 1000 {
	tx, err := db.Begin()
	i:=0
	for i < 1000 {
		species := allSpecies[rand.Intn(12)]
		country := allCountries[rand.Intn(11)]
		foliageColor := allColors[rand.Intn(6)]
		trunkColor := allColors[rand.Intn(6)]
		height := fmt.Sprintf("%d", rand.Intn(70))
		width := fmt.Sprintf("%d", rand.Intn(50))
		count := fmt.Sprintf("%d", rand.Intn(1000))
		leavyint := rand.Intn(2)
		leavy := "false"
		if leavyint == 1 {
			leavy = "true"
		}
		leavyint = rand.Intn(2)
		sick := "false"
		if leavyint == 1 {
			sick = "true"
		}		
		_, err = tx.Exec("insert into trees values(?,?,?,?,?,?,?,?,?)",
		species, country, leavy, foliageColor,  
		trunkColor, height, width, count, sick)
		if err != nil {
			log.Print("insert: " + err.Error())
			continue
		}
		i++
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	j++
	}
}

func hitTrees(db *sql.DB) {
	rows, err := db.Query("select * from trees where country = 'fr' and species = 'peuplier'")
	if err != nil {
		log.Fatal("select1: " + err.Error())
	}
	err = rows.Close()
	if err != nil {
		log.Fatal("close1: " + err.Error())
	}
/*
	rows, err = db.Query("select * from trees where country = 'fi' and species = 'bouleau' and foliagecolor = 'green'")
	if err != nil {
		log.Fatal("select2: " + err.Error())
	}
	err = rows.Close()
	if err != nil {
		log.Fatal("close2: " + err.Error())
	}
*/
}

func dropIndexes() {
	db, err := sql.Open("sqlite3", treesdb)
	if err != nil {
		log.Fatal("open: " + err.Error())
	}
	defer db.Close()
	_, err = db.Exec("drop index countryspec")
	if err != nil {
		log.Fatal("drop1: " + err.Error())
	}
	_, err = db.Exec("drop index countryspecfol")
	if err != nil {
		log.Fatal("drop2: " + err.Error())
	}
}

func createIndexes() {
	println("creating indexes")
	db, err := sql.Open("sqlite3", treesdb)
	if err != nil {
		log.Fatal("open: " + err.Error())
	}
	defer db.Close()
	_, err = db.Exec("create index 'countryspec' on trees('country', 'species')")
	if err != nil {
		log.Fatal("idx1: " + err.Error())
	}
	_, err = db.Exec("create index 'countryspecfol' on trees('country', 'species', 'foliagecolor')")
	if err != nil {
		log.Fatal("idx2: " + err.Error())
	}
}

var once sync.Once

func BenchmarkTrees(b *testing.B) {
	b.StopTimer()
	once.Do(createIndexes)
	b.StartTimer()
	db, err := sql.Open("sqlite3", treesdb)
	if err != nil {
		log.Fatal("open: " + err.Error())
	}
	defer db.Close()
	for i := 0; i < b.N; i++ {
		hitTrees(db)
	}
}
