package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"edge-cli/models"
)

type Client struct {
	baseURL   string
	token     string
	systemKey string
	http      *http.Client
}

func New(baseURL, token, systemKey string) *Client {
	return &Client{
		baseURL:   baseURL,
		token:     token,
		systemKey: systemKey,
		http:      &http.Client{},
	}
}

func (c *Client) do(method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("ClearBlade-DevToken", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// Authenticate hits the dev auth endpoint and returns the dev_token.
func Authenticate(baseURL, email, password string) (string, error) {
	payload := map[string]string{"email": email, "password": password}
	b, _ := json.Marshal(payload)

	resp, err := http.Post(baseURL+"/admin/auth", "application/json", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("authentication failed (HTTP %d): %s", resp.StatusCode, string(data))
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	token, ok := result["dev_token"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("no dev_token in response: %s", string(data))
	}
	return token, nil
}

// ListServices returns all deployed service metadata for the system.
func (c *Client) ListServices() ([]models.DBCodeMeta, error) {
	data, err := c.do("GET", "/codeadmin/v/2/codemeta/"+c.systemKey, nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code []models.DBCodeMeta `json:"code"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Code, nil
}

// GetService returns metadata for a single service by name.
func (c *Client) GetService(name string) (*models.DBCodeMeta, error) {
	services, err := c.ListServices()
	if err != nil {
		return nil, err
	}
	for _, svc := range services {
		if svc.Name == name {
			return &svc, nil
		}
	}
	return nil, fmt.Errorf("service %q not found", name)
}

// ListRunning returns all currently running service instances for the system.
func (c *Client) ListRunning() (map[string]models.RunningServiceInfo, error) {
	data, err := c.do("GET", "/codeadmin/v/3/running/"+c.systemKey, nil)
	if err != nil {
		return nil, err
	}
	var running map[string]models.RunningServiceInfo
	if err := json.Unmarshal(data, &running); err != nil {
		return nil, err
	}
	return running, nil
}

// GetLogs returns stored log runs for a service.
func (c *Client) GetLogs(serviceName string) ([]models.LegacyLogUnit, error) {
	data, err := c.do("GET", "/codeadmin/v/2/logs/"+c.systemKey+"/"+serviceName, nil)
	if err != nil {
		return nil, err
	}
	var logs []models.LegacyLogUnit
	if err := json.Unmarshal(data, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// StopInstance kills a specific running service instance by its request ID.
func (c *Client) StopInstance(instanceID string, timeoutSeconds int) error {
	body := map[string]any{"id": instanceID, "timeout": timeoutSeconds}
	_, err := c.do("DELETE", "/codeadmin/v/3/running/"+c.systemKey, body)
	return err
}

// StartService triggers execution of a service by name with optional params.
func (c *Client) StartService(name string, params map[string]any) error {
	if params == nil {
		params = map[string]any{}
	}
	_, err := c.do("POST", "/api/v/1/code/"+c.systemKey+"/"+name, params)
	return err
}

// GetLogByID returns the log for a specific service run.
func (c *Client) GetLogByID(serviceName, logID string) (*models.LegacyLogUnit, error) {
	data, err := c.do("GET", "/codeadmin/v/2/logs/"+c.systemKey+"/"+serviceName+"/"+logID, nil)
	if err != nil {
		return nil, err
	}
	var log models.LegacyLogUnit
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, err
	}
	return &log, nil
}
