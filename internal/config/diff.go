package config

import "strings"

// DiffLine is one line of a line diff; Op is " ", "+" or "-".
type DiffLine struct {
	Op   string `json:"op"`
	Text string `json:"text"`
}

// Diff returns a line-by-line diff from a to b using a longest common
// subsequence, which is plenty for config files of a few hundred lines.
func Diff(a, b string) []DiffLine {
	x, y := splitLines(a), splitLines(b)
	lcs := make([][]int, len(x)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(y)+1)
	}
	for i := len(x) - 1; i >= 0; i-- {
		for j := len(y) - 1; j >= 0; j-- {
			if x[i] == y[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var out []DiffLine
	i, j := 0, 0
	for i < len(x) && j < len(y) {
		switch {
		case x[i] == y[j]:
			out = append(out, DiffLine{" ", x[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, DiffLine{"-", x[i]})
			i++
		default:
			out = append(out, DiffLine{"+", y[j]})
			j++
		}
	}
	for ; i < len(x); i++ {
		out = append(out, DiffLine{"-", x[i]})
	}
	for ; j < len(y); j++ {
		out = append(out, DiffLine{"+", y[j]})
	}
	return out
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}
