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

func usage() {
	fmt.Println(`Usage: jirae COMMENT_URL
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
	// TODO: Add option to edit description.

	commentURL, err := commentURL()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Invalid comment URL:", err)
		os.Exit(2)
	}
	commentText, err := getComment(commentURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not retrieve comment:", err)
		os.Exit(2)
	}
	if commentText, err = getEditedText(commentText); err != nil {
		fmt.Fprintln(os.Stderr, "Could not get edited text:", err)
		os.Exit(2)
	}
	if userConfirmsSubmit() {
		err = updateCommentText(commentURL, commentText)
		if err != nil {
			fmt.Println("The comment was not updated. This new text is discarded:")
			fmt.Println(commentText)
			fmt.Fprintln(os.Stderr, "Could not update comment:", err)
			os.Exit(2)
		}
	} else {
		fmt.Println("The comment was not updated. This new text is discarded:")
		fmt.Println(commentText)
	}
}

func commentURL() (string, error) {
	re := regexp.MustCompile(
		`(https://.*\.atlassian\.net)/browse/([^?]+)\?focusedCommentId=([0-9]+)`)
	p := re.FindStringSubmatch(os.Args[1])
	if len(p) != 4 {
		return "", fmt.Errorf("invalid comment URL '%s'", os.Args[1])
	}
	return fmt.Sprintf("%s/rest/api/2/issue/%s/comment/%s", p[1], p[2], p[3]), nil
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
	err = json.NewDecoder(resp.Body).Decode(&commentJSON)
	if err != nil {
		return "", err
	}
	return commentJSON.Body, nil
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
	fmt.Print("Submit updated comment? [y/N]:")
	var input string
	fmt.Scanln(&input)
	return input == "y"
}

func updateCommentText(commentURL, commentText string) error {
	update := make(map[string]string)
	update["body"] = commentText
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
