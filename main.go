package main

import (
  "fmt"
	"bufio"
	"os"
	"log"
	"strconv"
	"time"
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

	fmt.Println("amount: ", amount)
	fmt.Println("date: ", date)
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
	default:
		log.Fatal("Invalid date format: " + str)
	}
	
	t, err = time.Parse("20060102", full)

	if err != nil {
		log.Fatal(err)
	}

	return t
}
