// Package main ...
package main

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/go-rod/rod/lib/utils"
)

func main() {
	rename()
}

func rename() {
	wg := sync.WaitGroup{}
	for _, p := range cmd("gopls -remote auto workspace_symbol .C.got") {
		pType := repose(p, -1, 4)

		cmd("gopls -remote auto rename -w " + pType + " T")

		wg.Add(1)
		go func() {
			for _, p := range cmd("gopls -remote auto references " + pType) {
				if strings.Contains(p, "definitions_test.go") {
					continue
				}

				pRef := repose(p, 0, -2)
				cmd("gopls -remote auto rename -w " + pRef + " t")
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

var regPos = regexp.MustCompile(`^(.+?):(\d+):(\d+)-\d+`)

func repose(raw string, lineOffset, columnOffset int) string {
	ms := regPos.FindStringSubmatch(raw)

	if ms == nil {
		log.Println("doesn't match", raw)
		return ""
	}

	p := ms[1]

	line, err := strconv.ParseInt(ms[2], 10, 64)
	utils.E(err)

	col, err := strconv.ParseInt(ms[3], 10, 64)
	utils.E(err)

	return fmt.Sprintf("%s:%d:%d", p, int(line)+lineOffset, int(col)+columnOffset)
}

func cmd(c string) []string {
	fmt.Println(c)
	args := strings.Split(c, " ")
	b, _ := exec.Command(args[0], args[1:]...).CombinedOutput()
	fmt.Println(string(b))
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}
