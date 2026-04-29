package main

import (
  "fmt"
	"bufio"
	"os"
)

func main() {
  fmt.Println("Hello World")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	fmt.Println("input: ", scanner.Text())
}
