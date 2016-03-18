package tests

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/opentable/sous/cli"
	"github.com/samsalisbury/semv"
	"github.com/xrash/smetrics"
)

type (
	// Terminal is a test harness for the sous CLI, providing easy
	// introspection into its inputs and outputs.
	Terminal struct {
		Sous *cli.Sous
		*cli.CLI
		Stdout, Stderr Output
		T              *testing.T
	}
	// Output allows inspection of output streams from the Terminal.
	Output struct {
		Name   string
		Buffer *bytes.Buffer
		T      *testing.T
	}
)

func NewTerminal(t *testing.T) *Terminal {
	out := Output{"stdout", &bytes.Buffer{}, t}
	err := Output{"stderr", &bytes.Buffer{}, t}
	return &Terminal{
		&cli.Sous{Version: semv.MustParse("0-test")},
		&cli.CLI{
			OutWriter: out.Buffer,
			ErrWriter: err.Buffer,
		},
		out, err, t,
	}
}

// RunCommand takes a command line, turns it into args, and passes it to a CLI
// which is pre-populated with a fresh *cli.Sous command, OutWriter and
// ErrWriter, both mapped to Outputs for interrogation.
//
// Note: This cannot cope with arguments containing spaces, even if they are
// surrounded by quotes. We should add this feature if we need it.
func (t *Terminal) RunCommand(commandline string) {
	args := strings.Split(commandline, " ")
	t.CLI.Invoke(t.Sous, args)
}

func (out Output) String() string { return out.Buffer.String() }

func (out Output) Lines() []string { return strings.Split(out.String(), "\n") }

func (out Output) LinesContaining(s string) []string {
	lines := []string{}
	for _, l := range out.Lines() {
		if strings.Contains(l, s) {
			lines = append(lines, l)
		}
	}
	return lines
}

func (out Output) NumLines() int {
	return strings.Count(out.String(), "\n")
}

func (out Output) HasLineMatching(s string) bool {
	for _, l := range out.Lines() {
		if l == s {
			return true
		}
	}
	return false
}

func (out Output) ShouldHaveExactLine(s string) {
	if out.HasLineMatching(s) {
		return
	}
	hint := out.similarLineHint(s)
	out.T.Errorf("expected %s to have exact line %q%s", out.Name, s, hint)
}

func (out Output) ShouldHaveLineContaining(s string) {
	for _, line := range out.Lines() {
		if strings.Contains(line, s) {
			return
		}
	}
	hint := out.similarLineHint(s)
	out.T.Errorf("expected %s to have line containing %q%s", out.Name, s, hint)
}

func (out Output) ShouldHaveNumLines(expected int) {
	actual := out.NumLines()
	if actual != expected {
		out.T.Errorf("%s has %d lines; want %d", out.Name, actual, expected)
	}
}

func (out Output) similarLineHint(s string) string {
	similar, i, goodMatch := out.MostSimilarLineTo(s)
	if !goodMatch {
		return ""
	}
	// we 1-index command output, "line 1" makes more sense than "line 0"
	i++
	return fmt.Sprintf("\nHowever, line %d looks similar: %q", i, similar)
}

// MostSimilarLineTo returns the most similar line in the output to the given
// string, if any of them have a JaroWinkler score >0.1. It returns the string
// (or empty), the index of that line, and a bool indicating if the score was
// greater than 0.1
func (out Output) MostSimilarLineTo(s string) (
	winner string, index int, goodMatch bool) {
	index = -1
	if s == "" {
		return
	}
	max := 0.0
	for i, l := range out.Lines() {
		score := smetrics.JaroWinkler(l, s, 0.7, 4)
		if score > max {
			winner = l
			index = i
			max = score
		}
	}
	return winner, index, max > 0.1
}