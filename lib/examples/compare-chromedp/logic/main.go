package main

import (
	"fmt"
	"github.com/ysmood/rod"
	"log"
	"strings"
	"time"
)

func main() {
	browser := rod.New().Connect()

	res, err := listAwesomeGoProjects(browser, "Selenium and browser control tools.")
	if err != nil {
		log.Fatalf("could not list awesome go projects: %v", err)
	}

	for k, v := range res {
		log.Printf("project %s (%s): '%s'", k, v.URL, v.Description)
	}
}

// projectDesc contains a url, description for a project.
type projectDesc struct {
	URL, Description string
}

// listAwesomeGoProjects is the highest level logic for browsing to the
// awesome-go page, finding the specified section sect, and retrieving the
// associated projects from the page.
func listAwesomeGoProjects(browser *rod.Browser, sect string) (map[string]projectDesc, error) {
	page := browser.Timeout(time.Second * 15).Page("https://github.com/avelino/awesome-go")

	sel := fmt.Sprintf(`//p[text()[contains(., '%s')]]`, sect)

	page.ElementX(sel).WaitVisible()

	sib := sel + `/following-sibling::ul/li`

	projects := page.ElementsX(sib + `/child::a/text()`)

	linksAndDescriptions := page.ElementsX(sib + `/child::node()`)

	if 2*len(projects) != len(linksAndDescriptions) {
		return nil, fmt.Errorf("projects and links and descriptions lengths do not match (2*%d != %d)", len(projects), len(linksAndDescriptions))
	}

	res := make(map[string]projectDesc)
	for i := 0; i < len(projects); i++ {
		res[projects[i].Describe().NodeValue] = projectDesc{
			URL:         linksAndDescriptions[2*i].Eval("() => this.href").Raw,
			Description: strings.TrimPrefix(strings.TrimSpace(linksAndDescriptions[2*i+1].Describe().NodeValue), "- "),
		}
	}

	return res, nil
}
