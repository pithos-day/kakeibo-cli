package main

import (
  "fmt"
	"bufio"
	"os"
	"log"
	"strconv"
	"time"
	"crypto/md5"
	"path/filepath"
	"database/sql"
	"golang.org/x/term"
	_ "github.com/mattn/go-sqlite3"
)

const version string = "1.0.0"
var args int

func main() {
	args = len(os.Args)
	if (args < 2) {
		commandHelp()
		return
	}

	var db *sql.DB
	var err error
	db, err = openDB()

	if (err != nil) {
		log.Fatal(err)
	}

	defer db.Close()

	switch os.Args[1] {
	case "help":
		commandHelp()
	case "version":
		commandVersion()
	case "expense":
		commandAddRecord(db, "EXPENSE")
	case "income":
		commandAddRecord(db, "INCOME")
	default:
		fmt.Println("Unknown command: ", os.Args[1])
	}
}

func openDB() (*sql.DB, error) {
	var home string
	var err error
	home, err = os.UserHomeDir()

	if (err != nil) {
		return nil, err
	}

	var dbPath string = filepath.Join(home, ".local", "share", "kakeibo-cli", "data.db")

	err = os.MkdirAll(filepath.Dir(dbPath), 0755)
	if (err != nil) {
		return nil, err
	}

	var db *sql.DB
	db, err = sql.Open("sqlite3", dbPath)
	if (err != nil) {
		return nil, err
	}

	var query string = `
	CREATE TABLE IF NOT EXISTS records (
		id TEXT PRIMARY KEY,
		amount INTEGER NOT NULL,
		date TEXT NOT NULL,
		memo TEXT,
		record_type TEXT NOT NULL
	);
	`
	_, err = db.Exec(query)

	if (err != nil) {
		return nil, err
	}

	return db, nil
}

func insertRecord(db *sql.DB, r Record) error {
	var err error
		_, err = db.Exec(
		`INSERT INTO records (id, amount, date, memo, record_type)
		 VALUES (?, ?, ?, ?, ?)`,
		r.ID, r.Amount, r.Date, r.Memo, r.RecordType,
	)

	return err
}

func commandHelp() {
	fmt.Println(`Usage:

	kakeibo-cli <command> [arguments]

The commands are:

	help		show this message
	version		show kakeibo-cli version`)
}

func commandVersion() {
	fmt.Println("kakeibo-cli v" + version)
}

func commandAddRecord(db *sql.DB, record_type string) {
	var amount int
	var date time.Time
	var memo string

	var sc *bufio.Scanner = bufio.NewScanner(os.Stdin)

	if (args < 3) {
		fmt.Print("Input Amount: ")
		sc.Scan()
		amount = mustInt(sc.Text())
	} else {
		amount = mustInt(os.Args[2])
	}

	if (args < 4) {
		fmt.Print("Input Date: ")
		sc.Scan()
		date = mustDate(sc.Text())
	} else {
		date = mustDate(os.Args[3])
	}

	if (args < 5) {
		fmt.Print("Input Memo: ")
		sc.Scan()
		memo = sc.Text()
	} else {
		memo = os.Args[4]
	}

	fmt.Println("Amount:", amount, ", Date:", date.Format("2006-01-02"), ", Memo:", memo, ", Type:" , record_type)
	if (checkYes()) {
		var r Record = Record{
			ID: generateHash(string(amount) + date.Format("2006-01-02") + memo + record_type + string(time.Now().UnixNano())),
			Amount: amount,
			Date: date.Format("2006-01-02"),
			Memo: memo,
			RecordType: record_type,
		}

		var err error = insertRecord(db, r)

		if (err != nil) {
			log.Fatal(err)	
		}

		fmt.Println("Successfully added record: ", r.ID)
	} else {
		return
	}
}

func mustInt(str string) int {
	var n int
	var err error

	n, err = strconv.Atoi(str)

	if (err != nil) {
		log.Fatal(err)
	}

	return n
}

func mustDate(str string) time.Time {
	var t time.Time
	var err error

	var full string
	switch len(str) {
	case 8:
		// YYYYMMDD
		full = str
	case 4:
		// MMDD
		var year int = time.Now().Year()
		full = fmt.Sprintf("%04d%s", year, str)
	case 1:
		if (str == "t") {
			full = time.Now().Format("20060102")
		} else if (str == "y") {
			full = time.Now().AddDate(0, 0, -1).Format("20060102")
		} else {
			log.Fatal("Inavlid date alias: " + str)
		}
	default:
		log.Fatal("Invalid date format: " + str)
	}
	
	t, err = time.ParseInLocation("20060102", full, time.Now().Location())

	if (err != nil) {
		log.Fatal(err)
	}

	return t
}

func checkYes() bool {
	var oldState *term.State
	var err error

	oldState, err = term.MakeRaw(int(os.Stdin.Fd()))

	if (err != nil) {
		log.Fatal(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	fmt.Print("Sure? [y/n]: ")

	var b []byte = make([]byte, 1)
	os.Stdin.Read(b)

	fmt.Print(string(b[0]))
	term.Restore(int(os.Stdin.Fd()), oldState)
	fmt.Println()

	switch b[0] {
	case 'y', 'Y':
		return true
	default:
		return false
	}
}

func generateHash(str string) string {
	var b []byte = []byte(str)
	var full string = fmt.Sprintf("%x", md5.Sum(b))
	return full[0:8]
}

type Record struct {
	ID     string
	Amount int
	Date   string
	Memo   string
	RecordType   string
}
