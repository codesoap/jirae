package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var (
	user  string
	token string
	cFlag bool
	fFlag string
)

type commentJSON struct {
	Body string
}

type issueJSON struct {
	Fields struct {
		Description string
	}
}

func usage() {
	fmt.Println(`Usage:
	jirae COMMENT_URL
	jirae ISSUE_URL
	jirae -c [-f] ISSUE_URL

Options:
	-c  Create a new comment for the issue with the given URL.
	-f  Additional fields to set in the request. E.g.:
	    {"visibility": {"type": "role", "value": "Admins"}}

The following environment variables need to be set:
	EDITOR
	JIRA_USER
	JIRA_TOKEN`)
	os.Exit(1)
}

func init() {
	flag.Usage = usage
	flag.BoolVar(&cFlag, "c", false, "Create a new comment.")
	flag.StringVar(&fFlag, "f", "{}",
		"JSON containing additional fields when creating comments.")
	flag.Parse()
	var err error
	if os.Getenv("EDITOR") == "" {
		err = fmt.Errorf("EDITOR environment variable is not set")
	} else if user = os.Getenv("JIRA_USER"); user == "" {
		err = fmt.Errorf("JIRA_USER environment variable is not set")
	} else if token = os.Getenv("JIRA_TOKEN"); token == "" {
		err = fmt.Errorf("JIRA_TOKEN environment variable is not set")
	} else if len(flag.Args()) != 1 {
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Fatal error:", err)
		os.Exit(1)
	}
}

func main() {
	// API documentation is located at
	// https://developer.atlassian.com/cloud/jira/platform/rest/v2/intro/

	var text, commentURL, issueURL string
	var err error
	if !cFlag {
		commentURL, err = readCommentURLArgument()
	}
	if cFlag || err != nil {
		if issueURL, err = readIssueURLArgument(); err != nil {
			fmt.Fprintln(os.Stderr, "Could not understand given argument:", err)
			os.Exit(2)
		}
	}
	if commentURL != "" {
		text, err = getComment(commentURL)
	} else if !cFlag {
		text, err = getIssue(issueURL)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not retrieve text:", err)
		os.Exit(2)
	}
	if text, err = getEditedText(text); err != nil {
		fmt.Fprintln(os.Stderr, "Could not get edited text:", err)
		os.Exit(2)
	}
	if userConfirmsSubmit() {
		if commentURL != "" {
			err = updateCommentText(commentURL, text)
		} else if !cFlag {
			err = updateIssueText(issueURL, text)
		} else {
			err = createComment(issueURL, text)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "Could not submit text:", err)
			fmt.Println("The text was not submitted. This new text is discarded:")
			fmt.Println(text)
			os.Exit(2)
		}
	} else {
		fmt.Println("The text was not submitted. This new text is discarded:")
		fmt.Println(text)
	}
}

func readCommentURLArgument() (string, error) {
	re := regexp.MustCompile(
		`^(https://.*\.atlassian\.net)/browse/([^?]+)\?focusedCommentId=([0-9]+)$`)
	p := re.FindStringSubmatch(flag.Args()[0])
	if len(p) != 4 {
		return "", fmt.Errorf("invalid comment URL '%s'", flag.Args()[0])
	}
	return fmt.Sprintf("%s/rest/api/2/issue/%s/comment/%s", p[1], p[2], p[3]), nil
}

func readIssueURLArgument() (string, error) {
	re := regexp.MustCompile(
		`^(https://.*\.atlassian\.net)/browse/([^?/]+)$`)
	p := re.FindStringSubmatch(flag.Args()[0])
	if len(p) != 3 {
		return "", fmt.Errorf("invalid issue URL '%s'", flag.Args()[0])
	}
	return fmt.Sprintf("%s/rest/api/2/issue/%s", p[1], p[2]), nil
}

func getComment(commentURL string) (string, error) {
	req, err := http.NewRequest("GET", commentURL, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(user, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var commentJSON commentJSON
	if err = json.NewDecoder(resp.Body).Decode(&commentJSON); err != nil {
		return "", err
	}
	return commentJSON.Body, nil
}

func getIssue(issueURL string) (string, error) {
	req, err := http.NewRequest("GET", issueURL, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(user, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var issueJSON issueJSON
	if err = json.NewDecoder(resp.Body).Decode(&issueJSON); err != nil {
		return "", err
	}
	return issueJSON.Fields.Description, nil
}

func getEditedText(text string) (string, error) {
	f, err := os.CreateTemp("", "jirae")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	if _, err := io.WriteString(f, text); err != nil {
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	editorCmd := exec.Command(os.Getenv("EDITOR"), f.Name())
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	if err = editorCmd.Run(); err != nil {
		return "", err
	}
	f, err = os.Open(f.Name())
	if err != nil {
		return "", err
	}
	defer f.Close()
	newText, err := io.ReadAll(f)
	return strings.TrimSpace(string(newText)), err
}

func userConfirmsSubmit() bool {
	fmt.Print("Submit updated text? [y/N]:")
	var input string
	fmt.Scanln(&input)
	return input == "y"
}

func updateCommentText(commentURL, text string) error {
	update := make(map[string]string)
	update["body"] = text
	s, err := json.Marshal(update)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", commentURL, bytes.NewBuffer(s))
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, token)
	req.Header.Set("Content-Type", "application/json")
	return doRequestAndCheckResponse(req)
}

func updateIssueText(issueURL, text string) error {
	update := make(map[string]any)
	fields := make(map[string]string)
	fields["description"] = text
	update["fields"] = fields
	s, err := json.Marshal(update)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", issueURL, bytes.NewBuffer(s))
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, token)
	req.Header.Set("Content-Type", "application/json")
	return doRequestAndCheckResponse(req)
}

func createComment(issueURL, text string) error {
	fields := make(map[string]any)
	if err := json.Unmarshal([]byte(fFlag), &fields); err != nil {
		return err
	}
	fields["body"] = text
	s, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", issueURL+"/comment", bytes.NewBuffer(s))
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, token)
	req.Header.Set("Content-Type", "application/json")
	return doRequestAndCheckResponse(req)
}

// doRequestAndCheckResponse executes the given request and returns an
// error if the request either returned an error or resulted in a non 2xx
// response code.
func doRequestAndCheckResponse(req *http.Request) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Could not read body of unexpected response: %w", err)
		}
		return fmt.Errorf("Got non-2xx response %d: %s", resp.StatusCode, body)
	}
	return nil
}
