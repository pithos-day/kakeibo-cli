package main

import (
  "fmt"
	"os"
	"errors"
	"strings"
	"strconv"
	"time"
	"crypto/md5"
	"path/filepath"
	"database/sql"
	"github.com/peterh/liner"
	_ "github.com/mattn/go-sqlite3"
)

const version string = "1.0.0"
const unit string = "円"

var line *liner.State
var args []string

func main() {
	args = os.Args
	if (len(args) < 2) {
		commandHelp()
		return
	}

	var db *sql.DB
	var err error
	db, err = openDB()

	if (err != nil) {
		fmt.Println(err)
		return
	}

	defer db.Close()

	line = liner.NewLiner()
	line.SetCtrlCAborts(true)
	defer line.Close()
	
	err = executeCommand(db)

	if (err != nil) {
		fmt.Println(err)
		return
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

func executeCommand(db *sql.DB) error {
	switch args[1] {
	case "help":
		commandHelp()
		return nil
	case "version":
		commandVersion()
		return nil
	case "repl":
		if (args[0] != "repl") {
			commandRepl(db)
		} else {
			fmt.Println("Can't execute repl in repl")
		}
		return nil
	case "show":
		return commandShow(db)
	case "stats":
		return commandStats(db)
	case "delete":
		return commandDeleteRecord(db)
	case "expense":
		return commandAddRecord(db, "EXPENSE")
	case "income":
		return commandAddRecord(db, "INCOME")
	default:
		fmt.Println("Unknown command: ", args[1])
		return nil
	}
}

var commands []string = []string{
	"help", "version", "show", "stats",
	"expense", "income", "delete", "exit",
}

func replCompleter(input string) []string {
	var res []string
	var cmd string
	for _, cmd = range commands {
		if strings.HasPrefix(cmd, input) {
			res = append(res, cmd)
		}
	}
	return res
}

func commandHelp() {
	
	if (len(args) > 2) {
		var input string = args[2]
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

		} else if (input == "stats") {
	fmt.Println(`Usage:

  kakeibo-cli stats [mode] [year]

Mode:

  all        Net balance (income - expense) by month
  expense    Expenses by month
  income     Income by month

Arguments:

  year       YYYY (e.g. 2026, default: current year)

Examples:

  kakeibo-cli stats
  kakeibo-cli stats expense`)
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
		} else if (input == "delete") {
				fmt.Println(`Usage:

	kakeibo-cli delete [id]

Arguments:

	id    	Hash id for record

Examples:

	kakeibo-cli delete 1cb193f6`)
		} else {
			fmt.Println("Invalid help topic: " + input)
		}
	} else {

	fmt.Println(`Usage:

	kakeibo-cli <command> [arguments]

The commands are:

	help		show this message
	version		show kakeibo-cli version
	repl        start interactive mode (type 'exit' to quit)
	show		show records
	stats		show monthly statistics
	expense		add expense record
	income		add income record
	delete		delete record

For details, run: kakeibo-cli help <command>`)
	}
}

func commandVersion() {
	fmt.Println("kakeibo-cli v" + version)
}

func commandRepl(db *sql.DB) {
	line.SetCompleter(replCompleter)
	for {
		var input string
		var err error
		input, err = line.Prompt("> ")

		if (err != nil) {
			break
		}

		if input == "exit" {
			break
		}

		var cmds []string = append([]string{"repl"}, strings.Fields(input)...)
		args = cmds

		if (1 < len(args)) {
			line.SetCompleter(nil)
			err = executeCommand(db)
			if (err != nil) {
				fmt.Println(err)
			}
			line.AppendHistory(input)
			line.SetCompleter(replCompleter)
		}
	}

}

func commandShow(db *sql.DB) error {
	var input string
	var err error

	if (len(args) < 3) {
		input, err = line.Prompt("Input date: ")

		if (err != nil) {
			return err
		}
	} else {
		input = args[2]
	}

	var rows *sql.Rows

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
		var args1 []any
		cond, args1 = buildDateQuery(input)

		var query string = fmt.Sprintf(`
		SELECT id, amount, date, memo, record_type
		FROM records
		WHERE %s
		ORDER BY date DESC
		`, cond)

		rows, err = db.Query(query, args1...)
	}

	if (err != nil) {
		return err
	}
	defer rows.Close()
	

	err = printRows(rows)

	if (err != nil) {
		return err
	}
	return nil
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
	fmt.Printf("[%s] %s%d%s %s (%s)\n", r.Date, t, r.Amount, unit, r.Memo, r.ID)
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

func commandStats(db *sql.DB) error {
	var input string
	var yyyy string
	var month int

	if (len(args) < 3) {
		input = "all"
	} else {
		input = args[2]
		if (input != "all" && input != "expense" && input != "income") {
			return errors.New("Invalid stats mode: " + input)
		}
	}

	var mode int = 0
	if (input == "expense") {
		mode = -1
	} else if (input == "income") {
		mode = 1
	}

	if (len(args) > 3) {
		if (len(args[3]) == 4) {
			yyyy = args[3]
			month = 12
		} else {
			return errors.New("Invalid year: " + args[3])
		}
	} else {
		yyyy = time.Now().Format("2006")
		month = int(time.Now().Month())
	}
	var rows *sql.Rows
	var err error
	var msgs []string = []string{}
	var gsums []int = []int{}
	var gsum_max int = 0
	var sum_total int = 0
	var i int = 1
	for ; i <= month; i++ {
		var mm string = fmt.Sprintf("%02d", i)
		var cond string
		var args1 []any
		cond, args1 = buildDateQuery(yyyy + mm)

		var query string = fmt.Sprintf(`
		SELECT id, amount, date, memo, record_type
		FROM records
		WHERE %s
		ORDER BY date DESC
		`, cond)

		rows, err = db.Query(query, args1...)

		if (err != nil) {
			return err
		}
		defer rows.Close()

		var sum int
		sum, err = getSumByRows(rows, mode)

		if (err != nil) {
			return err
		}

		sum_total += sum

		var gsum int = sum
		var t string = ""
		if (mode == 0) {
			gsum = min(gsum, 0)
			t = addSign(sum)
		} else {
			t = strconv.Itoa(abs(sum))
		}
		gsum = abs(gsum)
		gsum_max = max(gsum_max, gsum)
		gsums = append(gsums, gsum)

		msgs = append(msgs, fmt.Sprintf("[%s-%s] %s", yyyy, mm, t))
	}

	var average int = sum_total/month

	fmt.Printf("[%s-01 ~ %s-%s] Total: %s, Average: %s\n\n", yyyy, yyyy, fmt.Sprintf("%02d", month), addSign(sum_total), addSign(average))

	i = 0

	for ; i < month; i++ {
		var bar string
		var ratio int
		if (gsum_max == 0) {
			ratio = 1
		} else {
			ratio = max(ceilDiv(gsums[i]*40,gsum_max),1)
		}
		bar = strings.Repeat("\u2588", ratio) + strings.Repeat("\u2591", 40-ratio)
		fmt.Println(msgs[i])
		fmt.Println(bar)
	}
	return nil
}

func addSign(n int) string {
	var t string = ""
	switch {
	case n == 0:
    	t = "\u00B1"
	case n > 0:
    	t = "+"
	}
	return t + strconv.Itoa(n)
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

func abs(x int) int {
	if (x >= 0) {
		return x
	}
	return -x
}

func ceilDiv(a int,b int) int {
	return (a+b-1)/b
}

func commandDeleteRecord(db *sql.DB) error {
	var input string
	var err error

	if (len(args) < 3) {
		input, err = line.Prompt("Input ID: ")

		if (err != nil) {
			return err
		}
	} else {
		input = args[2]
	}

	var r Record

	r, err = getRecordByID(db, input)
	if (err != nil) {
		return err
	}

	printRecord(r)

	if (checkYes()) {
		err = deleteRecord(db, input)
		if (err != nil) {
			return err
		}

		fmt.Println("Successfully removed record: ", input)
	}
	return nil
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

func commandAddRecord(db *sql.DB, record_type string) error {
	var input string
	var amount int
	var date time.Time
	var memo string
	var err error

	if (len(args) < 3) {
		input, err = line.Prompt("Input Amount: ")
		if (err != nil) {
			return err
		}
		amount, err = strconv.Atoi(input)
		if (err != nil) {
			return err
		}
	} else {
		amount, err = strconv.Atoi(args[2])
		if (err != nil) {
			return err
		}
	}

	if (amount < 1) {
		return errors.New("Invalid amount: " + strconv.Itoa(amount))
	}

	if (len(args) < 4) {
		input, err = line.Prompt("Input Date: ")
		if (err != nil) {
			return err
		}
		date, err = parseDate(input)
		if (err != nil) {
			return err
		}
	} else {
		date, err = parseDate(args[3])
		if (err != nil) {
			return err
		}
	}

	if (len(args) < 5) {
		memo, err = line.Prompt("Input Memo: ")
		if (err != nil) {
			return err
		}
	} else {
		memo = args[4]
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

		err = insertRecord(db, r)

		if (err != nil) {
			return err	
		}

		fmt.Println("Successfully added record: ", r.ID)
	} else {
		return nil
	}
	return nil
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

func parseDate(str string) (time.Time, error) {
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
			return t, errors.New("Inavlid date alias: " + str)
		}
	default:
		return t, errors.New("Invalid date format: " + str)
	}
	
	t, err = time.ParseInLocation("20060102", full, time.Now().Location())

	if (err != nil) {
		return t, err
	}

	return t, nil
}

func checkYes() bool {
	input, err := line.Prompt("Sure? [y/n]: ")
	if err != nil {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
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
