// Package main provides a tool to fetch requests and attachments from the ZenGRC API.
//
// OBJECTIVE:
// This client manages authentication and low-level HTTP interactions with the ZenGRC API.
// It handles API endpoint construction, request signing (Basic Auth), JSON decoding,
// and file downloading with support for optimized connection pooling via a custom Transport.
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// --- API Endpoints and Configuration ---

// API endpoint paths
const (
	requestsPath           = "/api/v2/requests"
	requestDetailsPath     = "/api/v2/requests/%d"
	requestAttachmentsPath = "/api/v2/requests/%d/attachments"
	downloadFilePath       = "/api/v2/requests/%d/files/%d"
)

// --- Core Client and Initialization ---

// Client is a client for the ZenGRC API. It manages all interactions with the API.
type Client struct {
	apiURL     string
	token      string
	httpClient *http.Client
}

// NewClient creates a new ZenGRC API client with an optimized HTTP client.
// It configures a custom Transport with connection pooling (MaxIdleConns) and
// a 60-second request timeout.
func NewClient(apiURL, token string) *Client {
	// Configure a custom transport to optimize connection pooling and reuse.
	transport := &http.Transport{
		MaxIdleConns:    10,               // Max idle connections to keep open.
		IdleConnTimeout: 30 * time.Second, // Timeout for idle connections.
	}

	return &Client{
		apiURL: apiURL,
		token:  token,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   60 * time.Second, // Set a timeout for HTTP requests.
		},
	}
}

// --- Data Structures (ZenGRC API Schema) ---
// These structs are designed to match the JSON responses from the ZenGRC API,
// based on the provided swagger-v3.json specification.

// PersonInfo represents a person's basic information.
type PersonInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// AuditInfo represents basic audit information.
type AuditInfo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// CustomAttrValue represents a custom attribute value.
type CustomAttrValue struct {
	ID    int         `json:"id"`
	Title string      `json:"title"`
	Value interface{} `json:"value"`
}

// DetailsLinks represents the links for an object.
type DetailsLinks struct {
	Self struct {
		Href string `json:"href"`
	} `json:"self"`
}

// ControlInfo represents basic control information.
type ControlInfo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// IssueInfo represents basic issue information.
type IssueInfo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// ProgramInfo represents basic program information.
type ProgramInfo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// RequestMapped represents the mapped objects for a request.
type RequestMapped struct {
	Controls []ControlInfo `json:"controls"`
	Issues   []IssueInfo   `json:"issues"`
	Programs []ProgramInfo `json:"programs"`
}

// ReviewerStatus represents the status of a reviewer.
type ReviewerStatus struct {
	Reviewer PersonInfo `json:"reviewer"`
	Status   string     `json:"status"`
}

// Request represents a ZenGRC request object, containing its full metadata.
type Request struct {
	ID               int                        `json:"id"`
	Title            string                     `json:"title"`
	Code             string                     `json:"code"`
	Assignees        []PersonInfo               `json:"assignees"`
	Audit            AuditInfo                  `json:"audit"`
	CreatedAt        string                     `json:"created_at"`
	CustomAttributes map[string]CustomAttrValue `json:"custom_attributes"`
	Description      *string                    `json:"description"`
	DueDate          *string                    `json:"due_date"`
	Links            DetailsLinks               `json:"links"`
	Mapped           RequestMapped              `json:"mapped"`
	Notes            *string                    `json:"notes"`
	NotifyAssignee   *bool                      `json:"notify_assignee"`
	Requesters       []PersonInfo               `json:"requesters"`
	Reviewers        []ReviewerStatus           `json:"reviewers"`
	StartDate        string                     `json:"start_date"`
	Status           string                     `json:"status"`
	Tags             []string                   `json:"tags"`
	Test             *string                    `json:"test"`
	Type             string                     `json:"type"`
	UpdatedAt        string                     `json:"updated_at"`
	Verifiers        []PersonInfo               `json:"verifiers"`
}

// File represents a file attachment.
type File struct {
	DocumentID int    `json:"document_id"`
	Name       string `json:"name"`
	UploadedAt string `json:"uploaded_at"`
}

// RequestListResponse is the response from the API when listing requests.
type RequestListResponse struct {
	Data  []Request `json:"data"`
	Links struct {
		Next struct {
			Href string `json:"href"`
		} `json:"next"`
	} `json:"links"`
}

// AttachmentListResponse is the response from the API when listing attachments.
type AttachmentListResponse struct {
	Data struct {
		Files []File `json:"files"`
	} `json:"data"`
}

// --- Internal Request Handling ---

// newRequest creates a new HTTP request with the necessary headers for the ZenGRC API.
func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s%s", c.apiURL, path)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", basicAuth(c.token))
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// do executes an HTTP request and decodes the JSON response into the provided interface.
// It handles response body closure, non-200 status code errors, and JSON decoding.
func (c *Client) do(req *http.Request, v interface{}) error {
	// #nosec G704
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status: %s, body: %s", resp.Status, string(bodyBytes))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return err
		}
	}
	return nil
}

// --- Public API Methods ---

// GetRequestDetails retrieves the details of a single request.
func (c *Client) GetRequestDetails(requestID int) (*Request, error) {
	path := fmt.Sprintf(requestDetailsPath, requestID)
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var request Request
	if err := c.do(req, &request); err != nil {
		return nil, err
	}

	return &request, nil
}

// GetRequests retrieves a list of requests, handling pagination via the cursor.
// If the cursor is empty, it starts from the first page. The cursor from the API
// response is expected to be a full URL path.
func (c *Client) GetRequests(cursor string) (*RequestListResponse, error) {
	path := requestsPath
	if cursor != "" {
		path = cursor // The cursor from the API response is a full path.
	}

	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp RequestListResponse
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetAttachments retrieves the attachments for a given request.
func (c *Client) GetAttachments(requestID int) ([]File, error) {
	path := fmt.Sprintf(requestAttachmentsPath, requestID)
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp AttachmentListResponse
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}

	return resp.Data.Files, nil
}

// DownloadAttachment downloads a single attachment to the specified output directory.
// It includes a check to prevent overwriting existing files unless the overwrite flag is true.
func (c *Client) DownloadAttachment(requestID int, attachment File, outputDir string, overwrite bool) error {
	filePath := filepath.Join(outputDir, attachment.Name)

	// If overwrite is false, check if the file already exists.
	if !overwrite {
		if _, err := os.Stat(filePath); err == nil {
			fmt.Printf("File %s already exists. Skipping.\n", filePath)
			return nil
		}
	}

	path := fmt.Sprintf(downloadFilePath, requestID, attachment.DocumentID)
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return err
	}

	// #nosec G704
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status: %s, body: %s", resp.Status, string(bodyBytes))
	}

	// Create the output file.
	// #nosec G304
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); err != nil {
			log.Printf("Error closing file %s: %v", filePath, err)
		}
	}()

	// Copy the response body to the file.
	_, err = io.Copy(out, resp.Body)
	return err
}

// basicAuth returns a base64 encoded string for Basic Authentication.
func basicAuth(token string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(token))
}
