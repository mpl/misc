package misc

import (
	"database/sql"
	"log"
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
}

func BenchmarkCachedCdms(b *testing.B) {
	for i := 0; i < b.N; i++ {
		hitCDMS()
	}
}
