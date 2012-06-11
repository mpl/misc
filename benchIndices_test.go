package misc

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
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

func createTrees(indexed bool, fromScratch bool) {
	if fromScratch {
		err := os.Remove(treesdb)
		if err != nil {
			log.Fatal(err)
		}
	}
	db, err := sql.Open("sqlite3", treesdb)
	if err != nil {
		log.Fatal("open: " + err.Error())
	}
	defer db.Close()
	query := "create table trees (" +
		"'species' text, 'country' text, 'leavy' bool, 'foliagecolor' text," +
		"'trunkcolor' text, 'height' int, 'width' int, 'count' int, 'sick' bool," +
		" constraint speccountry unique (species, country))"
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
	i:=0
	for i < 100 {
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
		_, err = db.Exec("insert into trees values(?,?,?,?,?,?,?,?,?)",
		species, country, leavy, foliageColor,  
		trunkColor, height, width, count, sick)
		if err != nil {
			continue
//			log.Fatal("insert: " + err.Error())
		}
		i++
	}
}

func BenchmarkTrees(b *testing.B) {
//	b.StopTimer()
	createTrees(true, true)
//	b.StartTimer()
	for i := 0; i < b.N; i++ {
		println("booya")
	}
}
