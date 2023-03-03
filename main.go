package main

import (
	"bytes"
	"encoding/json"
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
	fmt.Println(`Usage: jirae ISSUE_OR_COMMENT_URL
The following environment variables need to be set:
- EDITOR
- JIRA_USER
- JIRA_TOKEN`)
	os.Exit(1)
}

func init() {
	var err error
	if os.Getenv("EDITOR") == "" {
		err = fmt.Errorf("EDITOR environment variable is not set")
	} else if user = os.Getenv("JIRA_USER"); user == "" {
		err = fmt.Errorf("JIRA_USER environment variable is not set")
	} else if token = os.Getenv("JIRA_TOKEN"); token == "" {
		err = fmt.Errorf("JIRA_TOKEN environment variable is not set")
	} else if len(os.Args) != 2 {
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Fatal error:", err)
		os.Exit(1)
	}
}

func main() {
	// TODO: Add option to add new comment.

	var text, commentURL, issueURL string
	var err error
	if commentURL, err = readCommentURLArgument(); err == nil {
		text, err = getComment(commentURL)
	} else {
		if issueURL, err = readIssueURLArgument(); err != nil {
			fmt.Fprintln(os.Stderr, "Could not understand given argument:", err)
			os.Exit(2)
		}
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
		} else {
			err = updateIssueText(issueURL, text)
		}
		if err != nil {
			fmt.Println("The text was not updated. This new text is discarded:")
			fmt.Println(text)
			fmt.Fprintln(os.Stderr, "Could not update text:", err)
			os.Exit(2)
		}
	} else {
		fmt.Println("The text was not updated. This new text is discarded:")
		fmt.Println(text)
	}
}

func readCommentURLArgument() (string, error) {
	re := regexp.MustCompile(
		`(https://.*\.atlassian\.net)/browse/([^?]+)\?focusedCommentId=([0-9]+)`)
	p := re.FindStringSubmatch(os.Args[1])
	if len(p) != 4 {
		return "", fmt.Errorf("invalid comment URL '%s'", os.Args[1])
	}
	return fmt.Sprintf("%s/rest/api/2/issue/%s/comment/%s", p[1], p[2], p[3]), nil
}

func readIssueURLArgument() (string, error) {
	re := regexp.MustCompile(
		`(https://.*\.atlassian\.net)/browse/([^?/]+)`)
	p := re.FindStringSubmatch(os.Args[1])
	if len(p) != 3 {
		return "", fmt.Errorf("invalid issue URL '%s'", os.Args[1])
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
	_, err = http.DefaultClient.Do(req)
	return err
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
	_, err = http.DefaultClient.Do(req)
	return err
}
