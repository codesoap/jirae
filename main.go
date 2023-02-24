package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

var (
	url   string
	user  string
	token string
)

type commentsJSON struct {
	Comments []commentJSON
}

type commentJSON struct {
	ID   string
	Body string
}

func usage() {
	fmt.Println(`Usage: jirae ISSUE_NUMBER [COMMENT_ID]
The following environment variables need to be set:
- EDITOR
- JIRA_URL
- JIRA_USER
- JIRA_TOKEN`)
	os.Exit(1)
}

func init() {
	var err error
	if os.Getenv("EDITOR") == "" {
		err = fmt.Errorf("EDITOR environment variable is not set")
	} else if url = strings.TrimRight(os.Getenv("JIRA_URL"), "/"); url == "" {
		err = fmt.Errorf("JIRA_URL environment variable is not set")
	} else if !strings.HasPrefix(url, "https://") {
		err = fmt.Errorf("JIRA_URL does not begin with https://")
	} else if user = os.Getenv("JIRA_USER"); user == "" {
		err = fmt.Errorf("JIRA_USER environment variable is not set")
	} else if token = os.Getenv("JIRA_TOKEN"); token == "" {
		err = fmt.Errorf("JIRA_TOKEN environment variable is not set")
	} else if len(os.Args) != 2 && len(os.Args) != 3 {
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

	issueID := os.Args[1]
	var commentID string
	var commentText string
	var err error
	if len(os.Args) == 3 {
		commentID = os.Args[2]
		commentText, err = getComment(issueID, commentID)
	} else {
		commentID, commentText, err = getLastComment(issueID)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not retrieve comment:", err)
		os.Exit(2)
	}
	if commentText, err = getEditedText(commentText); err != nil {
		fmt.Fprintln(os.Stderr, "Could not get edited text:", err)
		os.Exit(2)
	}
	if userConfirmsSubmit() {
		err = updateCommentText(issueID, commentID, commentText)
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

func getLastComment(issueID string) (string, string, error) {
	commentsURL := fmt.Sprintf("%s/rest/api/2/issue/%s/comment", url, issueID)
	req, err := http.NewRequest("GET", commentsURL, nil)
	if err != nil {
		return "", "", err
	}
	req.SetBasicAuth(user, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	var commentsJSON commentsJSON
	err = json.NewDecoder(resp.Body).Decode(&commentsJSON)
	if err != nil {
		return "", "", err
	}
	if len(commentsJSON.Comments) == 0 {
		return "", "", fmt.Errorf("no comments found on the given ticket")
	}
	lastComment := commentsJSON.Comments[len(commentsJSON.Comments)-1]
	return lastComment.ID, lastComment.Body, nil
}

func getComment(issueID, commentID string) (string, error) {
	commentsURL := fmt.Sprintf("%s/rest/api/2/issue/%s/comment/%s", url, issueID, commentID)
	req, err := http.NewRequest("GET", commentsURL, nil)
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

func updateCommentText(issueID, commentID, commentText string) error {
	update := make(map[string]string)
	update["body"] = commentText
	commentsURL := fmt.Sprintf("%s/rest/api/2/issue/%s/comment/%s", url, issueID, commentID)
	s, err := json.Marshal(update)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", commentsURL, bytes.NewBuffer(s))
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, token)
	req.Header.Set("Content-Type", "application/json")
	_, err = http.DefaultClient.Do(req)
	return err
}
