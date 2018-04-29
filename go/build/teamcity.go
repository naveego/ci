package build

import (
	"fmt"
	"log"
	"os"
	"regexp"
)

func RunningOnTeamCity() bool {
	_, ok := os.LookupEnv("TEAMCITY_VERSION")
	return ok
}

func SetTeamCityBuildNumber(version, buildNumber string) {
	if RunningOnTeamCity() {
		fmt.Printf(`##teamcity[buildNumber '%s Build %s']`, version, buildNumber)
	}
}

func SetTeamCityParameter(name, value string) {
	if RunningOnTeamCity() {
		fmt.Printf("##teamcity[setParameter name='%s' value='%s']", name, ciEscape(value))
	}
}

// CIBuildProblem reports an error in a way TeamCity can understand.
// If err is nil, this method does nothing.
// This method returns the error to support chaining.
func CIBuildProblem(err error) error {
	if err != nil {
		CIMessage("buildProblem", map[string]string{
			"description": err.Error(),
		})
	}
	return err
}

func CIProgress(message string) {
	CIMessage("progressMessage", message)
}

// CIMessage writes a message out in a format TeamCity can understand.
// The data parameter can be a string or a map[string]string.
func CIMessage(messageType string, data interface{}) {
	if RunningOnTeamCity() {
		message := "##teamcity[" + messageType

		switch d := data.(type) {
		case string:
			escaped := ciEscape(d)
			message += fmt.Sprintf(" '%s'", escaped)
		case map[string]string:
			for k, v := range d {
				escaped := ciEscape(v)
				message += fmt.Sprintf(" %s='%s'", k, escaped)
			}
		}
		message += "]"
		log.Println(message)
	} else {
		log.Printf("%s: %#v", messageType, data)
	}
}

var ciEscaper = regexp.MustCompile(`(['\n\r\|\]\[])`)

func ciEscape(s string) string {
	return ciEscaper.ReplaceAllString(s, "|$1")
}
