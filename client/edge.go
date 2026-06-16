package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"edge-cli/models"
)

type Client struct {
	baseURL   string
	token     string
	systemKey string
	edgeName  string // non-empty = proxy mode
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

// NewProxy creates a client that routes requests through the platform proxy to a named edge.
func NewProxy(platformURL, token, systemKey, edgeName string) *Client {
	return &Client{
		baseURL:   platformURL,
		token:     token,
		systemKey: systemKey,
		edgeName:  edgeName,
		http:      &http.Client{},
	}
}

// EdgeName returns the edge name if in proxy mode, otherwise empty string.
func (c *Client) EdgeName() string { return c.edgeName }

// ConnectionLabel returns a human-readable description of the connection target.
func (c *Client) ConnectionLabel() string {
	if c.edgeName != "" {
		return "proxy → " + c.baseURL + "  edge: " + c.edgeName
	}
	return "direct → " + c.baseURL
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
	if c.edgeName != "" {
		req.Header.Set("Clearblade-Edge", c.edgeName)
		req.Header.Set("Clearblade-Systemkey", c.systemKey)
	}

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

// AuthenticateViaProxy authenticates against an edge through the platform proxy,
// returning an edge-local dev token. The platform token is used only to satisfy
// the proxy routing headers — the actual auth is handled by the edge.
func AuthenticateViaProxy(platformURL, platformToken, systemKey, edgeName string) (string, error) {
	// We need the user's credentials to auth against the edge.
	// Re-use the same email/password by calling /admin/auth on the edge via proxy.
	// The platform token carries the email — fetch it, then auth through the proxy.
	//
	// Simplest approach: proxy a token-less auth using the platform token to get
	// the dev email, then prompt... but we already have credentials from the
	// platform login. Instead, call /admin/auth on the edge using those same creds
	// by passing the platform token as a "hint" — the edge's /admin/auth requires
	// email+password so we need a second call with actual credentials.
	//
	// For now: make a direct proxy auth call using the platform token against the
	// edge's /api/v/1/code endpoint (DevAndUser) to check if the platform token
	// happens to work, then fall back to prompting.
	//
	// Actually the correct path: call /admin/auth on the edge via the proxy,
	// but that needs email+password. We'll re-use what was passed to Authenticate().
	// Since we don't store the password, we need to prompt again.
	// Return a sentinel error so the caller can show a credential prompt.
	return "", fmt.Errorf("edge requires separate login — use edge-cli auth login --url %s with proxy headers", platformURL)
}

// AuthenticateEdgeViaProxy authenticates directly against an edge through the platform proxy
// using explicit credentials, returning an edge-local dev token.
// /admin/auth is a Public endpoint — no DevToken header is sent to avoid the platform
// rejecting the request before proxying it.
func AuthenticateEdgeViaProxy(platformURL, platformToken, systemKey, edgeName, email, password string) (string, error) {
	payload := map[string]string{"email": email, "password": password}
	b, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", platformURL+"/admin/auth", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	// Do NOT set ClearBlade-DevToken here — /admin/auth is Public and some platform
	// versions validate the DevToken before the proxy check, causing a 400.
	req.Header.Set("Clearblade-Edge", edgeName)
	req.Header.Set("Clearblade-Systemkey", systemKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("edge auth failed (HTTP %d): %s", resp.StatusCode, string(data))
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	token, ok := result["dev_token"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("no dev_token in edge auth response: %s", string(data))
	}
	return token, nil
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

// GetServiceCode returns the source code of a service.
func (c *Client) GetServiceCode(name string) (string, error) {
	data, err := c.do("GET", "/api/v/1/code/"+c.systemKey+"/"+name, nil)
	if err != nil {
		return "", err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	code, _ := result["code"].(string)
	return code, nil
}

// ListEdges returns all edges for the system (platform only).
func (c *Client) ListEdges() ([]models.EdgeInfo, error) {
	data, err := c.do("GET", "/admin/edges/"+c.systemKey, nil)
	if err != nil {
		return nil, err
	}
	var edges []models.EdgeInfo
	if err := json.Unmarshal(data, &edges); err != nil {
		return nil, err
	}
	return edges, nil
}

// ListCollections returns all collections for the system.
func (c *Client) ListCollections() ([]models.CollectionInfo, error) {
	data, err := c.do("GET", "/api/v/3/allcollections/"+c.systemKey, nil)
	if err != nil {
		return nil, err
	}
	var cols []models.CollectionInfo
	if err := json.Unmarshal(data, &cols); err != nil {
		return nil, err
	}
	return cols, nil
}

// CollectionQueryOpts controls what QueryCollection returns.
type CollectionQueryOpts struct {
	SortBy   string
	SortDesc bool
	Limit    int
	Page     int
}

// QueryCollection fetches rows from a collection by name.
func (c *Client) QueryCollection(name string, opts CollectionQueryOpts) (*models.CollectionData, error) {
	pageSize := 25
	if opts.Limit > 0 {
		pageSize = opts.Limit
	}
	page := 1
	if opts.Page > 1 {
		page = opts.Page
	}

	q := map[string]any{
		"PAGESIZE": pageSize,
		"PAGENUM":  page,
	}
	if opts.SortBy != "" {
		dir := "ASC"
		if opts.SortDesc {
			dir = "DESC"
		}
		q["SORT"] = []map[string]any{{dir: opts.SortBy}}
	}

	qJSON, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}

	path := "/api/v/1/collection/" + c.systemKey + "/" + name + "?query=" + url.QueryEscape(string(qJSON))
	data, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result models.CollectionData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// LogQueryOpts controls what the v4 logs endpoint returns.
type LogQueryOpts struct {
	ServiceName     string
	Level           string
	Since           time.Time    // zero means no lower bound
	AfterTimeMicros int64        // exclusive lower bound in microseconds (used for follow polling)
	Limit           int
}

// GetLogsV4 queries the v4 code logs endpoint with rich filtering.
func (c *Client) GetLogsV4(opts LogQueryOpts) ([]models.LogEntry, error) {
	filters := [][]map[string]any{}

	if opts.ServiceName != "" {
		filters = append(filters, []map[string]any{{"EQ": map[string]any{"name": opts.ServiceName}}})
	}
	if opts.Level != "" {
		filters = append(filters, []map[string]any{{"EQ": map[string]any{"level": opts.Level}}})
	}

	var timeThreshold int64
	if opts.AfterTimeMicros > 0 {
		timeThreshold = opts.AfterTimeMicros
	} else if !opts.Since.IsZero() {
		timeThreshold = opts.Since.UnixMicro()
	}
	if timeThreshold > 0 {
		filters = append(filters, []map[string]any{{"GT": map[string]any{"time": timeThreshold}}})
	}

	pageSize := 50
	if opts.Limit > 0 {
		pageSize = opts.Limit
	}

	q := map[string]any{
		"SORT":     []map[string]any{{"ASC": "time"}},
		"PAGESIZE": pageSize,
		"PAGENUM":  1,
	}
	if len(filters) > 0 {
		q["FILTERS"] = filters
	}

	qJSON, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}

	path := "/api/v/4/" + c.systemKey + "/code/logs?query=" + url.QueryEscape(string(qJSON))
	data, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var entries []models.LogEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
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
