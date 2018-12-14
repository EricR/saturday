package encoding

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
)

func ParseDimacs(in io.Reader) ([][]int, error) {
	scanner := bufio.NewScanner(in)
	sentences := [][]int{}

	for scanner.Scan() {
		sentence := []int{}
		fields := bytes.Fields(scanner.Bytes())

		if len(fields) < 2 {
			continue
		}
		prefix := string(fields[0])

		if prefix == "c" || prefix == "p" {
			continue
		}
		for _, field := range fields[:len(fields)] {
			p, err := strconv.Atoi(string(field))
			if err != nil {
				return nil, err
			}
			if p != 0 {
				sentence = append(sentence, p)
			}
		}
		sentences = append(sentences, sentence)
	}
	return sentences, nil
}
