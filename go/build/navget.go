package build

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"archive/zip"
	"fmt"
	"log"

	"io/ioutil"
)

const packageFilePath = "./package.zip"

func NewNavgetClient() (NavgetClient, error) {
	var err error
	client := NavgetClient{}
	if client.Endpoint, err = getRequiredEnv("NAVGET_ENDPOINT"); err != nil {
		return client, err
	}
	if client.Token, err = getRequiredEnv("NAVGET_TOKEN"); err != nil {
		return client, err
	}
	return client, nil
}

func getRequiredEnv(names ...string) (string, error) {
	for _, name := range names {
		if val, ok := os.LookupEnv(name); ok {
			return val, nil
		}
	}

	return "", fmt.Errorf("one of %v must be defined as an environment variable", names)
}

type NavgetClient struct {
	Endpoint string
	Token    string
}

type NavgetParams struct {
	OS    string
	Arch  string
	Files []string
}

func (c NavgetClient) Upload(p NavgetParams) error {

	if err := c.executeCreate(p); err != nil {
		return fmt.Errorf("create failed: %s", err)
	}

	if err := c.executePublish(p); err != nil {
		return fmt.Errorf("upload failed: %s", err)
	}

	os.Remove(packageFilePath)

	return nil
}

func (c NavgetClient) executePublish(p NavgetParams) error {

	log.Printf("Uploading '%s'...", packageFilePath)

	endpoint := c.Endpoint
	url := fmt.Sprintf("%s/api/packages?os=%s&arch=%s", endpoint, p.OS, p.Arch)
	log.Printf("Uploading to '%s'...", url)

	token := c.Token

	file, err := os.Open(packageFilePath)
	defer file.Close()
	if err != nil {
		return fmt.Errorf("error reading package file: %s", err)
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(packageFilePath))
	if err != nil {
		return fmt.Errorf("couldn't create form file: %s", err)
	}
	_, err = io.Copy(part, file)

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("couldn't close form file: %s", err)
	}
	log.Printf("Upload size: %d bytes", body.Len())
	client := &http.Client{
		Timeout: time.Second * 30,
	}
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("couldn't create request: %s", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("couldn't upload file: %s", err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	json := buf.String()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error uploading - status=%v, response=%s", resp.StatusCode, json)
	}

	defer resp.Body.Close()
	log.Printf("Uploaded '%s' to '%s'.", packageFilePath, url)

	return nil
}

func (c NavgetClient) executeCreate(p NavgetParams) error {

	if _, err := os.Stat("manifest.json"); os.IsNotExist(err) {
		return fmt.Errorf("manifest.json file not found: %s", err)
	}

	log.Printf("Creating package at '%s'...", packageFilePath)

	files := append(p.Files, "manifest.json")

	written := map[string]bool{}

	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new zip archive.
	w := zip.NewWriter(buf)

	for _, file := range files {
		// don't write files twice
		if _, alreadyWritten := written[file]; alreadyWritten {
			continue
		}
		written[file] = true
		bytes, err := ioutil.ReadFile(file)
		if err != nil {
			return fmt.Errorf("error reading file '%s': %s", file, err)
		}
		f, err := w.Create(file)
		if err != nil {
			return err
		}
		_, err = f.Write(bytes)
		if err != nil {
			return err
		}
	}

	// Make sure to check the error on Close.
	err := w.Close()
	if err != nil {
		return err
	}

	ioutil.WriteFile(packageFilePath, buf.Bytes(), 0644)

	log.Printf("Created package at '%s'.", packageFilePath)
	return nil
}
