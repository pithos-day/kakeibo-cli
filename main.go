package main

import (
  "fmt"
	"bufio"
	"os"
	"log"
	"strconv"
	"time"
	"crypto/md5"

	"golang.org/x/term"
)

const version string = "1.0.0"
var args int

func main() {
	args = len(os.Args)
	if (args < 2) {
		commandHelp()
		return
	}

	switch os.Args[1] {
	case "help":
		commandHelp()
	case "version":
		commandVersion()
	case "expense":
		commandExpense()
	default:
		fmt.Println("Unknown command: ", os.Args[1])
	}
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

func commandExpense() {
	var amount int
	var date time.Time
	var memo string

	var sc *bufio.Scanner = bufio.NewScanner(os.Stdin)

	if (args < 3) {
		fmt.Print("Input Amount: ")
		sc.Scan()
		amount = convInt(sc.Text())
	} else {
		amount = convInt(os.Args[2])
	}

	if (args < 4) {
		fmt.Print("Input Date: ")
		sc.Scan()
		date = convDate(sc.Text())
	} else {
		date = convDate(os.Args[3])
	}

	if (args < 5) {
		fmt.Print("Input Memo: ")
		sc.Scan()
		memo = sc.Text()
	} else {
		memo = os.Args[4]
	}

	fmt.Println("Amount:", amount, ", Date:", date.Format("2006/01/02"), ", Memo:", memo)
	if (checkYes()) {
		fmt.Println("success: ", generateHash(string(amount) + date.Format("2006/01/02") + memo))
	} else {
		return
	}
}

func convInt(str string) int {
	var n int
	var err error

	n, err = strconv.Atoi(str)

	if err != nil {
		log.Fatal(err)
	}

	return n
}

func convDate(str string) time.Time {
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

	if err != nil {
		log.Fatal(err)
	}

	return t
}

func checkYes() bool {
	var oldState *term.State
	var err error

	oldState, err = term.MakeRaw(int(os.Stdin.Fd()))

	if err != nil {
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
