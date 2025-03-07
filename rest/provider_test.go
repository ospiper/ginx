package sentinel

import (
	"log"
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	var err error
	testDB, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"))
	if err != nil {
		log.Fatal(err)
	}
	code := m.Run()
	os.Exit(code)
}
