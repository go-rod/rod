package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

const mirror = "https://registry.npmmirror.com/-/binary/chromium-browser-snapshots/"

func main() {
	list := getList(mirror)

	revLists := [][]int{}
	for _, os := range list {
		revList := []int{}
		for _, s := range getList(mirror + os + "/") {
			rev, err := strconv.ParseInt(s, 10, 32)
			if err != nil {
				log.Fatal(err)
			}
			revList = append(revList, int(rev))
		}
		sort.Ints(revList)
		revLists = append(revLists, revList)
	}

	rev := largestCommonRevision(revLists)

	if rev < 969819 {
		utils.E(fmt.Errorf("cannot match version of the latest chromium from %s", mirror))
	}

	playwright := getFromPlaywright()

	out := utils.S(`// generated by "lib/launcher/revision"

package launcher

// RevisionDefault for chromium
const RevisionDefault = {{.default}}

// RevisionPlaywright for arm linux
const RevisionPlaywright = {{.playwright}}
`,
		"default", rev,
		"playwright", playwright,
	)

	utils.E(utils.OutputFile(filepath.FromSlash("lib/launcher/revision.go"), out))

}

func getList(path string) []string {
	res, err := http.Get(path)
	utils.E(err)
	defer func() { _ = res.Body.Close() }()

	var data interface{}
	err = json.NewDecoder(res.Body).Decode(&data)
	utils.E(err)

	list := data.([]interface{})

	names := []string{}
	for _, it := range list {
		name := it.(map[string]interface{})["name"].(string)
		names = append(names, strings.TrimRight(name, "/"))
	}

	return names
}

func largestCommonRevision(revLists [][]int) int {
	sort.Slice(revLists, func(i, j int) bool {
		return len(revLists[i]) < len(revLists[j])
	})

	shortest := revLists[0]

	for i := len(shortest) - 1; i >= 0; i-- {
		r := shortest[i]

		isCommon := true
		for i := 1; i < len(revLists); i++ {
			if !has(revLists[i], r) {
				isCommon = false
				break
			}
		}
		if isCommon {
			return r
		}
	}

	return 0
}

func has(list []int, i int) bool {
	index := sort.SearchInts(list, i)
	return index < len(list) && list[index] == i
}

func getFromPlaywright() int {
	pv := strings.TrimSpace(utils.ExecLine(false, "npm --no-update-notifier -s show playwright version"))
	out := fetch(fmt.Sprintf("https://raw.githubusercontent.com/microsoft/playwright/v%s/packages/playwright-core/browsers.json", pv))
	rev, err := strconv.ParseInt(gson.NewFrom(out).Get("browsers.0.revision").Str(), 10, 32)
	utils.E(err)
	return int(rev)
}

func fetch(u string) string {
	res, err := http.Get(u)
	utils.E(err)
	defer func() { _ = res.Body.Close() }()

	b, err := io.ReadAll(res.Body)
	utils.E(err)
	return string(b)
}
