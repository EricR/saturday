package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/ericr/saturday/config"
	"github.com/ericr/saturday/encoding"
	"github.com/ericr/saturday/solver"
	"os"
	"time"
)

func main() {
	conf := config.New()
	parseFlags(conf)

	sentences, err := readCNF(flag.Args()[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	sat := solver.New(conf)

	for _, clause := range sentences {
		sat.AddClause(clause)
	}
	conf.Logger.Printf("Starting Saturday %s solver", solver.Version())

	tStart := time.Now()
	models := solve(sat, conf)

	conf.Logger.Print("Finished solving")

	displayStats(sat, time.Now().Sub(tStart))

	if len(models) == 0 {
		fmt.Fprint(os.Stdout, "p UNSAT\n")
		os.Exit(3)
	}
	fmt.Fprint(os.Stdout, "p SAT\n")
	displayModels(models)
	os.Exit(0)
}

func solve(sat *solver.Solver, conf *config.Config) [][]int {
	if conf.Models > 1 {
		return sat.SolveMany([]int{}, conf.Models)
	}
	if sat.Solve([]int{}) {
		return [][]int{sat.Answer()}
	}
	return [][]int{}
}

func displayModels(models [][]int) {
	for _, model := range models {
		for _, p := range model {
			fmt.Fprintf(os.Stdout, "%d ", p)
		}
		fmt.Fprint(os.Stdout, "0\n")
	}
}

func displayStats(s *solver.Solver, t time.Duration) {
	fmt.Fprint(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Time Taken:    %fs\n", t.Seconds())
	fmt.Fprintf(os.Stderr, "Variables:     %d\n", s.NVars())
	fmt.Fprintf(os.Stderr, "Constraints:   %d\n", s.NConstrs())
	fmt.Fprintf(os.Stderr, "Conflicts:     %d\n", s.NConflicts())
	fmt.Fprintf(os.Stderr, "Propagations:  %d\n", s.NPropagations())
	fmt.Fprintf(os.Stderr, "Restarts:      %d\n", s.NRestarts())
	fmt.Fprintf(os.Stderr, "Decisions:     %d\n", s.NDecisions())
	fmt.Fprint(os.Stderr, "\n")
}

func parseFlags(c *config.Config) {
	flag.UintVar(&c.Models, "m", uint(1), "number of models to find")
	flag.Float64Var(&c.VarDecay, "decay-var", 0.95, "variable decay constant")
	flag.Float64Var(&c.VarDecay, "decay-cla", 0.999, "clause decay constant")
	flag.Usage = flagUsage
	flag.Parse()

	if len(os.Args) < 2 {
		flagUsage()
		os.Exit(2)
	}
}

func flagUsage() {
	fmt.Fprintf(os.Stderr, "Usage: saturday input.cnf [args]"+
		"\n\nValid Arguments:\n")
	flag.PrintDefaults()
}

func readCNF(path string) ([][]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if !isFile(path) {
		return nil, fmt.Errorf("open %s: not a readable file", path)
	}
	return encoding.ParseDimacs(bufio.NewReader(f))
}

func isFile(path string) bool {
	if fs, err := os.Stat(path); err == nil {
		if fs.Mode().IsRegular() {
			return true
		}
	}
	return false
}
