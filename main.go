package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
		created_at TEXT NOT NULL,
		category TEXT NOT NULL
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
	case "export":
		return commandTransfer(db, 0)
	case "import":
		return commandTransfer(db, 1)
	default:
		fmt.Println("Unknown command: ", args[1])
		return nil
	}
}

func commandHelp() {
	
	if (len(args) > 2) {
		var input string = args[2]
		if (input == "show") {
			fmt.Println(`Usage:

	kakeibo-cli show [filter]

Filter:

	latest		Latest 5 records
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
 
		kakeibo-cli stats [date] [mode] [filter]
 
Arguments:
 
	date		YYYY / YYYYMM / MM (default: current year)
			y	Alias for current year  (YYYY)
			m	Alias for current month (YYYYMM)

	mode		period      Statistics by period (default)
			category    Statistics by main category
			subcategory Statistics by subcategory
	filter		all         Net balance, income - expense (default)
			expense     Expenses only
			income      Income only
 
Examples:
 
	kakeibo-cli stats
	kakeibo-cli stats 2026
	kakeibo-cli stats 202601
	kakeibo-cli stats 2026 category
	kakeibo-cli stats 2026 period expense`)
		} else if (input == "expense" || input == "income") {
				fmt.Println(`Usage:

	kakeibo-cli ` + input + ` [amount] [date] [category] [memo]

Arguments:

	amount		Expense amount (e.g. 1000)
	date		Date (YYYYMMDD, MMDD, t=today, y=yesterday)
	category	Category (default: Misc, supports subcategory e.g. Food.Lunch)
	memo		Description of the record

Examples:
`)
			if (input == "expense") {
				fmt.Println("	kakeibo-cli expense 1850 20010203 Food.Lunch Lunch")
			} else {
				fmt.Println("	kakeibo-cli income 210000 0203 Salary")
			}
		} else if (input == "delete") {
				fmt.Println(`Usage:

	kakeibo-cli delete [id]

Arguments:

	id		Hash id for record

Examples:

	kakeibo-cli delete 1cb193f6`)
		} else if (input == "export") {
				fmt.Println(`Usage:
 
	kakeibo-cli export [path]
 
Arguments:
 
	path		Output file path (default: ~/Documents/exported_kakeibo_YYYYMMDD-{timestamp}.json)
 
Examples:
 
	kakeibo-cli export
	kakeibo-cli export /tmp/backup.json`)
		} else if (input == "import") {
				fmt.Println(`Usage:
 
	kakeibo-cli import [path]
 
Arguments:
 
	path		JSON file path to import (duplicate records are skipped)
 
Examples:
 
	kakeibo-cli import /tmp/backup.json`)
		} else {
			fmt.Println("Invalid help topic: " + input)
		}
	} else {

	fmt.Println(`Usage:

	kakeibo-cli <command> [arguments]

The commands are:

	help		show this message
	version		show kakeibo-cli version
	repl		start interactive mode (type 'exit' to quit)
	show		show records
	stats		show statistics
	expense		add expense record
	income		add income record
	delete		delete record
	export		export records to JSON
	import		import records from JSON

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
		SELECT id, amount, date, memo, record_type, category
		FROM records
		ORDER BY created_at DESC
		LIMIT 5
		`)
	} else if (input == "all") {
		rows, err = db.Query(`
		SELECT id, amount, date, memo, record_type, category
		FROM records
		ORDER BY date DESC
		`)
	} else {
		var cond string
		var args1 []any
		cond, args1 = buildDateQuery(input)

		var query string = fmt.Sprintf(`
		SELECT id, amount, date, memo, record_type, category
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
		err = rows.Scan(&r.ID, &r.Amount, &r.Date, &r.Memo, &r.RecordType, &r.Category)
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
	fmt.Printf("[%s] %s%d%s %s (%s) : %s\n", r.Date, t, r.Amount, unit, r.Memo, r.ID, r.Category)
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
	var date string
	var err error

	if (len(args) < 3) {
		date = time.Now().Format("2006")
	} else {
		var flag bool = false
		switch len(args[2]) {
		case 1:
			if (args[2] == "y") {
				date = time.Now().Format("2006")
			} else if (args[2] == "m") {
				date = time.Now().Format("200601")
			} else {
				flag = true
			}
		case 2:
			var mm int
			mm, err = strconv.Atoi(args[2])
			if (err != nil) {
				return err
			}

			if (0 < mm && mm < 13) {
				date = time.Now().Format("2006") + args[2]
			} else {
				flag = true
			}
		case 4:
			date = args[2]
		case 6:
			date = args[2]
		default:
			flag = true
		}

		if (flag) {
			return errors.New("Invalid date: " + args[2])
		}
	}

	var mode int = 0

	if (len(args) > 3) {
		switch args[3] {
		case "period":
			mode = 0
		case "category":
			mode = 1
		case "subcategory":
			mode = 2
		default:
			return errors.New("Invalid mode: " + args[3])
		}
	}

	var filter int

	if (len(args) < 5) {
		filter = 0
	} else {
		switch args[4] {
		case "all":
			filter = 0
		case "expense":
			filter = -1
		case "income":
			filter = 1
		default:
			return errors.New("Invalid filter: " + args[4])
		}
	}

	var recordSets []RecordSet = []RecordSet{}
	var label string

	if (mode == 0) {
		var i int = 1
		var i1 int
		var label1 string
		switch len(date) {
		case 4:
			label1 = date
			if (date == time.Now().Format("2006")) {
				i1 = int(time.Now().Month())
			} else {
				i1 = 12
			}
		case 6:
			label1 = date[:4] + "-" + date[4:]
			if (date == time.Now().Format("200601")) {
				i1 = time.Now().Day()
			} else {
				var year int
				year, err = strconv.Atoi(date[:4])
				if (err != nil) {
					return err
				}
				var month int
				month, err = strconv.Atoi(date[4:])
				if (err != nil) {
					return err
				}
				i1 = time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local).Day()
			}
		}

		label = fmt.Sprintf("[%s-01 ~ %s-%s]", label1, label1, fmt.Sprintf("%02d", i1))

		for ; i <= i1; i++ {
			var cond string
			var args1 []any
			cond, args1 = buildDateQuery(date + fmt.Sprintf("%02d", i))

			var query string = fmt.Sprintf(`
			SELECT id, amount, date, memo, record_type, category
			FROM records
			WHERE %s
			ORDER BY date DESC
			`, cond)

			var rows *sql.Rows
			rows, err = db.Query(query, args1...)

			if (err != nil) {
				return err
			}
			defer rows.Close()

			var rs RecordSet 
			rs, err = rowsToRecordSet(label1 + "-" + fmt.Sprintf("%02d", i), rows, filter)

			if (err != nil) {
				return err
			}
			recordSets = append(recordSets, rs)
		}
	} else if (mode == 1 || mode == 2) {
		label = "[Categories]"

		var cond string
		var args1 []any

		cond, args1 = buildDateQuery(date)

		var query string 

		if (mode == 1) {
			query = fmt.Sprintf(`
				SELECT DISTINCT
					CASE 
						WHEN INSTR(category, '.') > 0
						THEN SUBSTR(category, 1, INSTR(category, '.') - 1)
						ELSE category
					END AS main
				FROM records
				WHERE %s
				ORDER BY category ASC
			`, cond)
		} else {
			query = fmt.Sprintf(`
				SELECT DISTINCT category
				FROM records
				WHERE %s
				ORDER BY category ASC
			`, cond)
		}

		var rows *sql.Rows
		rows, err = db.Query(query, args1...)

		if err != nil {
			return err
		}

		defer rows.Close()

		var categories []string

		for rows.Next() {
			var category string
			err = rows.Scan(&category)
			if err != nil {
				return err
			}
			categories = append(categories, category)
		}

		var rows1 *sql.Rows
		var category string

		for _, category = range categories {
			var query1 string
			var args2 []any

			if mode == 1 {
				query1 = fmt.Sprintf(`
					SELECT id, amount, date, memo, record_type, category
					FROM records
					WHERE (category = ? OR category LIKE ?) AND %s
				`, cond)

				args2 = append([]any{category, category + ".%"}, args1...)
			} else {
				query1 = fmt.Sprintf(`
					SELECT id, amount, date, memo, record_type, category
					FROM records
					WHERE category = ? AND %s
				`, cond)

				args2 = append([]any{category}, args1...)
			}

			rows1, err = db.Query(query1, args2...)

			if err != nil {
				return err
			}

			var rs RecordSet
			rs, err = rowsToRecordSet(category, rows1, filter)
			if (err != nil) {
				return err
			}
			recordSets = append(recordSets, rs)
			rows1.Close()
		}
	}

	var total int = 0
	var maximum int = 1

	for _, r := range recordSets {
		total += r.Sum

		if (filter == 0) {
			maximum = max(maximum, abs(min(r.Sum,0)))
		} else {
			maximum = max(maximum, abs(r.Sum))
		}
	}

	fmt.Printf("%s Total: %s, Average: %s\n\n", label, sign(total), sign(total/max(len(recordSets),1)))

	for _, r := range recordSets {
		fmt.Println(r.Name, sign(r.Sum))

		var ratio int = 1
		if (maximum != 0) {
			var absSum int
			if (filter == 0) {
				absSum = abs(min(r.Sum, 0))
			} else {
				absSum = abs(r.Sum)
			}
			ratio = max(ceilDiv(absSum*40,maximum),1)
		}

		fmt.Println(strings.Repeat("\u2588", ratio) + strings.Repeat("\u2591", 40-ratio))
	}

	return nil
}

func rowsToRecordSet(label string, rows *sql.Rows, filter int) (RecordSet, error) {
	var recordSet RecordSet
	var records []Record
	var sum int

	var err error
	for rows.Next() {
		var r Record
		err = rows.Scan(&r.ID, &r.Amount, &r.Date, &r.Memo, &r.RecordType, &r.Category)
		if err != nil {
			return recordSet, err
		}
		records = append(records, r)
		
		if (filter < 1 && r.RecordType == "EXPENSE") {
			sum -= r.Amount
		} else if (filter > -1 && r.RecordType == "INCOME") {
			sum += r.Amount
		}
	}

	recordSet.Name = label
	recordSet.Records = records
	recordSet.Sum = sum

	return recordSet, nil
}

func sign(n int) string {
	var t string = ""
	switch {
	case n == 0:
			t = "\u00B1"
	case n > 0:
			t = "+"
	}
	return t + strconv.Itoa(n)
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

		fmt.Println("Successfully deleted record: ", input)
	}
	return nil
}

func getRecordByID(db *sql.DB, id string) (Record, error) {
	var r Record
	var err error
	err = db.QueryRow(`
		SELECT id, amount, date, memo, record_type, category
		FROM records
		WHERE id = ?
	`, id).Scan(
		&r.ID,
		&r.Amount,
		&r.Date,
		&r.Memo,
		&r.RecordType,
		&r.Category,
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
	var memo string = ""
	var category string
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
		category, err = line.Prompt("Input Category (Empty to Misc): ")
		if (err != nil) {
			return err
		}
	} else {
		category = args[4]
	}

	if (category == "") {
		category = "Misc"
	}

	if (len(args) > 5) {
		memo = args[5]
	}

	fmt.Printf("Amount: %d, Date: %s, Memo: %s, Type: %s, Category: %s\n", amount, date.Format("2006-01-02"), memo, record_type, category)
	if (checkYes()) {
		var r Record = Record{
			ID: generateHash(string(amount) + date.Format("2006-01-02") + memo + record_type + fmt.Sprintf("%d", time.Now().UnixNano())),
			Amount: amount,
			Date: date.Format("2006-01-02"),
			Memo: memo,
			RecordType: record_type,
			CreatedAt: fmt.Sprintf("%d", time.Now().Unix()),
			Category: category,
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
		`INSERT INTO records (id, amount, date, memo, record_type, created_at, category)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Amount, r.Date, r.Memo, r.RecordType, r.CreatedAt, r.Category,
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
	var input string
	var err error
	input, err = line.Prompt("Sure? [y/n]: ")
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

func commandTransfer(db *sql.DB, mode int) error {
	var path string
	var err error

	if (len(args) < 3) {
		if (mode == 0) {
			path, err = defaultExportPath()
		} else {
			path, err = line.Prompt("Input Path: ")
		}

		if (err != nil) {
			return err
		}
	} else {
		path = args[2]
	}
	
	if (mode == 0) {
		//export
		return exportJson(db, path)
	} else if (mode == 1) {
		//import
		return importJson(db, path)
	}
	return nil
}

func defaultExportPath() (string, error) {
	var home string
	var err error

	home, err = os.UserHomeDir()
	if (err != nil) {
		return "", err
	}

	var date string = time.Now().Format("20060102")
	var ts int64 = time.Now().Unix()

	var path string = filepath.Join(
		home,
		"Documents",
		fmt.Sprintf("exported_kakeibo_%s-%d.json", date, ts),
	)

	return path, nil
}

func exportJson(db *sql.DB, path string) error {
	var rows *sql.Rows
	var err error

	rows, err = db.Query(`
		SELECT id, amount, date, memo, record_type, created_at, category
		FROM records
	`)
	if (err != nil) {
		return err
	}
	defer rows.Close()

	var records []Record

	for rows.Next() {
		var r Record
		err = rows.Scan(&r.ID, &r.Amount, &r.Date, &r.Memo, &r.RecordType, &r.CreatedAt, &r.Category)
		if (err != nil) {
			return err
		}
		records = append(records, r)
	}

	var f *os.File
	f, err = os.Create(path)
	if (err != nil) {
		return err
	}
	defer f.Close()

	var enc *json.Encoder = json.NewEncoder(f)
	enc.SetIndent("", "  ")

	return enc.Encode(records)
}

func importJson(db *sql.DB, path string) error {
	var f *os.File
	var err error

	f, err = os.Open(path)
	if (err != nil) {
		return err
	}
	defer f.Close()

	var records []Record

	err = json.NewDecoder(f).Decode(&records)
	if (err != nil) {
		return err
	}

	for _, r := range records {
		_, err = db.Exec(`
			INSERT OR IGNORE INTO records
			(id, amount, date, memo, record_type, created_at, category)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, r.ID, r.Amount, r.Date, r.Memo, r.RecordType, r.CreatedAt, r.Category)

		if (err != nil) {
			return err
		}
	}

	return nil
}

func generateHash(str string) string {
	var b []byte = []byte(str)
	var full string = fmt.Sprintf("%x", md5.Sum(b))
	return full[0:8]
}

type RecordSet struct {
	Name string
	Records []Record
	Sum int
}

type Record struct {
	ID string
	Amount int
	Date string
	Memo string
	RecordType string
	CreatedAt string
	Category string
}
