package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
)

const (
	template = `package main

func main() {
	%s
}`
	tempFileDir    = "temp"
	tempFilePrefix = "snippet"
)

type GoPlaygroundResult struct {
	CompileErrors string `json:"compile_errors"`
	Output        string `json:"output"`
}

func (r *GoPlaygroundResult) GetOutput() string {
	return fmt.Sprintf("```%s```", r.Output)
}

type GoPlaygroundClient struct {
	Host    string
	DebugOn bool
}

func (c *GoPlaygroundClient) Printf(format string, v ...interface{}) {
	if c.DebugOn {
		log.Printf(format, v...)
	}
}

func (c *GoPlaygroundClient) CleanUpTempFile(f *os.File) {
	filename := f.Name()
	var err error

	err = f.Close()
	if err != nil {
		c.Printf("Error closing temp file (filename: %s)\n", filename)
	}

	go func() {
		if err = os.Remove(filename); err != nil {
			c.Printf("Error removing temp file (filename: %s): %v\n", filename, err)
		}
		c.Printf("Temp file removed (filename: %s)\n", filename)
	}()
}

func (c *GoPlaygroundClient) Format(input string) (string, error) {
	// Create temp file
	f, err := ioutil.TempFile(tempFileDir, tempFilePrefix)
	if err != nil {
		return "", err
	}
	defer c.CleanUpTempFile(f)

	// Write snippet to temp file
	snippet := fmt.Sprintf(template, input)
	_, err = f.WriteString(snippet)
	if err != nil {
		return "", err
	}
	tempFilename := f.Name()
	c.Printf("Snippet written to temp file: %s\n", tempFilename)

	// Run 'goimports', grab output from stdout
	output, err := exec.Command("goimports", tempFilename).Output()
	if err != nil {
		return "", err
	}

	formatted := string(output)
	c.Printf("Code formatted:\n%s\n", formatted)
	return formatted, nil
}

func (c *GoPlaygroundClient) Compile(code string) (*GoPlaygroundResult, error) {
	formatted, err := c.Format(code)
	if err != nil {
		return nil, err
	}

	resp, err := http.PostForm(c.Host, url.Values{"body": {formatted}})
	if err != nil {
		return nil, err
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			c.Printf("Error closing response body: %v\n", err)
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result GoPlaygroundResult
	if err = json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
