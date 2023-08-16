package main

import (
	"fmt"

	"github.com/mr-joshcrane/oracle"
)

func main() {
	question := "How much wood would a woodchuck chuck?"
	o := oracle.NewOracle()
	q, err := o.Ask(question)
	if err != nil {
		panic(err)
	}
	fmt.Println(q)
}
