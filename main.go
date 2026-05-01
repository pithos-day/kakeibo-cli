package main

import (
  "fmt"
	"bufio"
	"os"
	"log"
	"strings"
	"strconv"
	"time"
	"crypto/md5"
	"path/filepath"
	"database/sql"
	"golang.org/x/term"
	_ "github.com/mattn/go-sqlite3"
)

const version string = "1.0.0"
const unit string = "円"

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
	case "show":
		commandShow(db)
	case "stats":
		commandStats(db)
	case "delete":
		commandDeleteRecord(db)
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
		record_type TEXT NOT NULL,
		created_at TEXT
	);
	`
	_, err = db.Exec(query)

	if (err != nil) {
		return nil, err
	}

	return db, nil
}

func commandHelp() {
	
	if (args > 2) {
		var input string = os.Args[2]
		if (input == "show") {
			fmt.Println(`Usage:

	kakeibo-cli show [filter]

Filter:

	latest   	Latest 10 records
  	all		All records
  	YYYYMMDD	Specific date (20010203)
  	YYYYMM		Specific month (200102)
  	YYYY		Specific year (2001)
	MMDD		Day in current year (0203)
  	MM		Month in current year (02)
		t		Today
		y		Yesterday

Examples:

  kakeibo-cli show latest
  kakeibo-cli show 200102
  kakeibo-cli show 02
	kakeibo-cli show t`)

		} else if (input == "expense" || input == "income") {
				fmt.Println(`Usage:

	kakeibo-cli ` + input + ` [amount] [date] [memo]

Arguments:

	amount    	Expense amount (e.g. 1000)
  	date    	Date (YYYYMMDD, MMDD, t=today, y=yesterday)
  	memo    	Description of the record

Examples:
`)
			if (input == "expense") {
  			fmt.Println("	kakeibo-cli expense 1850 20010203 Lunch")
			} else {
				fmt.Println("	kakeibo-cli income 210000 0203 Salary")
			}
		} else {
			fmt.Println("Invalid help topic: " + input)
		}
	} else {

	fmt.Println(`Usage:

	kakeibo-cli <command> [arguments]

The commands are:

	help		show this message
	version		show kakeibo-cli version
	show		show records
	expense		add expense record
	income		add income record

For details, run: kakeibo-cli help <command>`)
	}
}

func commandVersion() {
	fmt.Println("kakeibo-cli v" + version)
}

func commandShow(db *sql.DB) {
	var input string

	if (args < 3) {
		var sc *bufio.Scanner = bufio.NewScanner(os.Stdin)
		fmt.Print("Input date: ")
		sc.Scan()
		input = sc.Text()
	} else {
		input = os.Args[2]
	}

	var rows *sql.Rows
	var err error

	if (input == "latest") {
		rows, err = db.Query(`
		SELECT id, amount, date, memo, record_type
		FROM records
		ORDER BY created_at DESC
		LIMIT 5
		`)
	} else if (input == "all") {
		rows, err = db.Query(`
		SELECT id, amount, date, memo, record_type
		FROM records
		ORDER BY date DESC
		`)
	} else {
		var cond string
		var args []any
		cond, args = buildDateQuery(input)

		var query string = fmt.Sprintf(`
		SELECT id, amount, date, memo, record_type
		FROM records
		WHERE %s
		ORDER BY date DESC
		`, cond)

		rows, err = db.Query(query, args...)
	}

	if (err != nil) {
		log.Fatal(err)
	}
	defer rows.Close()
	

	err = printRows(rows)

	if (err != nil) {
		log.Fatal(err)
	}
}

func printRows(rows *sql.Rows) error {
	for rows.Next() {
		var r Record
		var err error
		err = rows.Scan(&r.ID, &r.Amount, &r.Date, &r.Memo, &r.RecordType)
		if (err != nil) {
			return err
		}

		printRecord(r)
	}
	return nil
}

func printRecord(r Record) {
	var t string = "-"
	if r.RecordType == "INCOME" {
		t = "+"
	}
	fmt.Printf("[%s] %s %s%d%s %s\n", r.ID, r.Date, t, r.Amount, unit, r.Memo)
}

func buildDateQuery(input string) (string, []any) {
	var today string = time.Now().Format("20060102")

	switch len(input) {
	case 9:
		if (input == "yesterday") {
			return "REPLACE(date, '-', '') = ?", []any{time.Now().AddDate(0, 0, -1).Format("20060102")}
		}
	case 8:
		// YYYYMMDD
		return "REPLACE(date, '-', '') = ?", []any{input}
	case 6:
		// YYYYMM
		return "REPLACE(date, '-', '') LIKE ?", []any{input + "%"}
	case 5:
		if (input == "today") {
			return "REPLACE(date, '-', '') = ?", []any{today}
		}
	case 4:
		if (input[:2] > "12") {
			// YYYY
			return "REPLACE(date, '-', '') LIKE ?", []any{input + "%"}
		} else {
			// MMDD
			return "REPLACE(date, '-', '') = ?", []any{today[:4] + input}
		}
	case 2:
		// MM
		return "REPLACE(date, '-', '') LIKE ?", []any{today[:4] + input + "%"}
	case 1:
		if (input == "t") {
			return "REPLACE(date, '-', '') = ?", []any{today}
		} else if (input == "y") {
			return "REPLACE(date, '-', '') = ?", []any{time.Now().AddDate(0, 0, -1).Format("20060102")}
		}
	}
	return "0=1", nil
}

func commandStats(db *sql.DB) {
	var input string

	if (args < 3) {
		input = "all"
	} else {
		input = os.Args[2]
		if (input != "all" && input != "expense" && input != "income") {
			log.Fatal("Invalid stats mode: " + input)
		}
	}

	var mode int = 0
	if (input == "expense") {
		mode = -1
	} else if (input == "income") {
		mode = 1
	}

	var yyyy string = time.Now().Format("2006")
	var month int = int(time.Now().Month())
	var rows *sql.Rows
	var err error
	var msgs []string = []string{}
	var gsums []int = []int{}
	var gsum_total int = 0
	var i int = 1
	for ; i <= month; i++ {
		var mm string = fmt.Sprintf("%02d", i)
		var cond string
		var args []any
		cond, args = buildDateQuery(mm)

		var query string = fmt.Sprintf(`
		SELECT id, amount, date, memo, record_type
		FROM records
		WHERE %s
		ORDER BY date DESC
		`, cond)

		rows, err = db.Query(query, args...)

		if (err != nil) {
			log.Fatal(err)
		}
		defer rows.Close()

		var sum int
		sum, err = getSumByRows(rows, mode)

		if (err != nil) {
			log.Fatal(err)
		}

		var gsum int = sum
		var t string = ""
		if (mode == 0) {
			gsum = getMin(gsum, 0)
			switch {
			case sum == 0:
    		t = "\u00B1"
			case sum > 0:
    		t = "+"
			}
		} else {
			sum = getAbs(sum)
		}
		gsum = getAbs(gsum)
		gsum_total += gsum
		gsums = append(gsums, gsum)

		msgs = append(msgs, fmt.Sprintf("[%s-%s] %s%d", yyyy, mm, t, sum))
	}

	i = 0

	for ; i < month; i++ {
		fmt.Println(msgs[i])
		var ratio int = (gsums[i]*20+gsum_total-1)/gsum_total
		fmt.Println(strings.Repeat("\u2588", ratio) + strings.Repeat("\u2591", 20-ratio))
	}
}

func getSumByRows(rows *sql.Rows, mode int) (int, error) {
	// mode 0 = all, -1 = expense only, 1 = income only
	var result int = 0
	for rows.Next() {
		var r Record
		var err error
		err = rows.Scan(&r.ID, &r.Amount, &r.Date, &r.Memo, &r.RecordType)
		if err != nil {
			return result, err
		}

		if (mode < 1 && r.RecordType == "EXPENSE") {
			result -= r.Amount
		} else if (mode > -1 && r.RecordType == "INCOME") {
			result += r.Amount
		}
	}
	return result, nil
}

func getAbs(x int) int {
	if (x >= 0) {
		return x
	}
	return -x
}

func getMin(x int,m int) int {
	if (x < m) {
		return x
	}
	return m
}

func commandDeleteRecord(db *sql.DB) {
	var input string

	if (args < 3) {
		var sc *bufio.Scanner = bufio.NewScanner(os.Stdin)
		fmt.Print("Input ID: ")
		sc.Scan()
		input = sc.Text()
	} else {
		input = os.Args[2]
	}

	var r Record
	var err error

	r, err = getRecordByID(db, input)
	if (err != nil) {
		log.Fatal(err)
	}

	printRecord(r)

	if (checkYes()) {
		err = deleteRecord(db, input)
		if (err != nil) {
			log.Fatal(err)
		}

		fmt.Println("Successfully removed record: ", input)
	}
}

func getRecordByID(db *sql.DB, id string) (Record, error) {
	var r Record
	var err error

	err = db.QueryRow(`
		SELECT id, amount, date, memo, record_type
		FROM records
		WHERE id = ?
	`, id).Scan(
		&r.ID,
		&r.Amount,
		&r.Date,
		&r.Memo,
		&r.RecordType,
	)

	return r, err
}

func deleteRecord(db *sql.DB, id string) error {
	var err error
	_, err = db.Exec("DELETE FROM records WHERE id = ?", id)
	return err
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

	if (amount < 1) {
		log.Fatal("Invalid amount: ", amount)
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
			ID: generateHash(string(amount) + date.Format("2006-01-02") + memo + record_type + fmt.Sprintf("%d", time.Now().UnixNano())),
			Amount: amount,
			Date: date.Format("2006-01-02"),
			Memo: memo,
			RecordType: record_type,
			CreatedAt: fmt.Sprintf("%d", time.Now().Unix()),
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

func insertRecord(db *sql.DB, r Record) error {
	var err error
		_, err = db.Exec(
		`INSERT INTO records (id, amount, date, memo, record_type, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.Amount, r.Date, r.Memo, r.RecordType, r.CreatedAt,
	)

	return err
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
		if (str == "t" || str == "today") {
			full = time.Now().Format("20060102")
		} else if (str == "y" || str == "yesterday") {
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
	ID string
	Amount int
	Date string
	Memo string
	RecordType string
	CreatedAt string
}
