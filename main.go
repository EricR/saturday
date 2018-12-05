package main

import (
	// "flag"
	"fmt"
	"github.com/ericr/saturday/solver"
	"log"
	"os"
)

func main() {
	printBanner()

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	
	sat := solver.New(logger)
	sat.AddClause([]int{-1, -3, 5})
	sat.AddClause([]int{-1, -3, -5})

	if sat.Solve([]int{1}) {
		fmt.Println("\nSAT")

		for i, val := range sat.Answer() {
			fmt.Printf("%d = %t\n", i, val)
		}
	} else {
		fmt.Println("\nUNSAT")
	}
}

func printBanner() {
	fmt.Printf("Saturday Solver %s\n", solver.Version())
	fmt.Println("https://ericrafaloff.com/saturday")
	fmt.Println("")
}
