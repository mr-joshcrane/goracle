package main

import (
	"fmt"
	"os"

	"github.com/mr-joshcrane/oracle"
)

func main() {
	question := "How much wood would a woodchuck chuck?"
	o := oracle.NewOracle()
	q, err := o.Ask(question)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	fmt.Fprintln(os.Stdout, q)
}
