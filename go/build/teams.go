package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/magefile/mage/sh"
)

func NotifyTeams(name, version, errorMessage string) {

	var (
		msTeamsChannel string
		jiraURL        string
		branch         string
		ok             bool
		err            error
	)

	if msTeamsChannel, ok = os.LookupEnv("MSTEAMS_DEPLOY_CHANNEL"); !ok {
		fmt.Println("If you set the MSTEAMS_DEPLOY_CHANNEL env parameter we will notify when the upgrade is complete.")
		return
	}

	branch, err = GitBranch()
	if err != nil {
		return
	}

	gitLog, err := sh.Output("git", "log", "--pretty=format:'%h - %an, %ar : %s", "--graph", "--since=1.week")

	if err != nil {
		log.Printf("Error Collecting Log Info Teams: %s", err)
	}

	var message = ""
	if jiraURL, ok = os.LookupEnv("JIRA_URL"); ok {

		issueIDExp := regexp.MustCompile("([A-Z]{2,5}-[0-9]{0,5})")

		matches := issueIDExp.FindAllString(gitLog, -1)
		matchSet := map[string]bool{}

		if len(matches) > 0 {
			message = "<div><p>The following JIRA issues are related to this deployment:</p>"
			for _, match := range matches {
				if !matchSet[match] {
					message += fmt.Sprintf("<div><a href='%s/browse/%sm'>%s</a></div>", jiraURL, match, match)
					matchSet[match] = true
				}
			}
			message += "</div>"
		}

	}

	var title = "Deployed " + name + "@" + branch + " (Version: " + version + ") to "
	var color = "48A555"
	message += "<div><p>Commits on this branch in the past 7 days:</p><pre>" + gitLog + "</pre></div>"

	if errorMessage != "" {
		title = "FAILED to Deployed Pipeline-UI (Version: " + version + ") to "
		message = "<div>ERROR: " + errorMessage + "</div>"
		color = "FF0000"
	}

	if OnReleaseBranch() {
		title += " Test"
	} else if OnMasterBranch() {
		title += " Production"
	}

	deployMessage := map[string]string{
		"title":      title,
		"text":       message,
		"themeColor": color,
	}
	deployJSON, _ := json.Marshal(deployMessage)

	client := &http.Client{}

	_, err = client.Post(msTeamsChannel, "application/json", bytes.NewReader(deployJSON))
	if err != nil {
		log.Printf("WARNING: Could not notify MS Teams about deploy: %s", err)
		return
	}

	log.Printf("Notified MS Teams: %#v", deployMessage)
}
