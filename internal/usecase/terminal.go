package usecase

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

// Terminal is ...
type Terminal struct {
	scanner *bufio.Scanner
}

// NewTerminal ...
func NewTerminal() *Terminal {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)

	return &Terminal{
		scanner: scanner,
	}
}

// ValText ...
func (t *Terminal) ValText() (string, bool) {
	if t.scanner.Scan() {
		return t.scanner.Text(), true
	}

	return "", false
}

// ValBoolean ...
func (t *Terminal) ValBoolean() (bool, error) {
	res, ok := t.ValText()
	if ok {
		lower := strings.ToLower(res)
		if lower == "y" {
			return true, nil
		}

		if lower == "n" {
			return false, nil
		}
	}

	return false, errors.New("terminal just accept Y / N")
}
