package models

// DBCodeMeta mirrors the platform's service metadata response.
type DBCodeMeta struct {
	Uuid                string   `json:"uuid"`
	SystemKey           string   `json:"system_key"`
	Name                string   `json:"name"`
	Version             int      `json:"version"`
	ExecutionTimeout    int      `json:"execution_timeout"`
	Concurrency         int      `json:"concurrency"`
	LoggingEnabled      bool     `json:"logging_enabled"`
	LogTTLMinutes       int      `json:"log_ttl_minutes"`
	RunOnPlatform       bool     `json:"run_on_platform"`
	RunOnEdge           bool     `json:"run_on_edge"`
	EngineType          int      `json:"engine_type"`
	AutoBalance         bool     `json:"auto_balance"`
	AutoScale           bool     `json:"auto_scale"`
	MinScaleConcurrency int      `json:"min_scale_concurrency"`
	MaxScaleConcurrency int      `json:"max_scale_concurrency"`
	LogLevel            string   `json:"log_level"`
	Topics              []string `json:"topics"`
}

// HeapStatistics mirrors ServiceHeapStatistics.
type HeapStatistics struct {
	GoHeap HeapDetail `json:"go_heap"`
	JsHeap HeapDetail `json:"js_heap"`
	Error  string     `json:"error,omitempty"`
}

type HeapDetail struct {
	CurrentBytesAllocated uint64 `json:"CurrentBytesAllocated"`
	CurrentBytesOverhead  uint64 `json:"CurrentBytesOverhead"`
}

func (h *HeapStatistics) TotalBytesAllocated() uint64 {
	return h.GoHeap.CurrentBytesAllocated + h.JsHeap.CurrentBytesAllocated
}

// RunningServiceInfo mirrors the platform's running service response.
type RunningServiceInfo struct {
	ExecutingAs    string         `json:"ExecutingAs"`
	Started        int64          `json:"Started"`
	SystemKey      string         `json:"SystemKey"`
	CodeName       string         `json:"CodeName"`
	Node           string         `json:"Node"`
	NodeId         string         `json:"NodeId"`
	IsTerminating  bool           `json:"IsTerminating"`
	EngineType     int            `json:"EngineType"`
	HeapStatistics HeapStatistics `json:"HeapStatistics"`
}

// LegacyLogUnit mirrors the platform's log response.
type LegacyLogUnit struct {
	ID        string `json:"id"`
	Log       string `json:"log"`
	Time      string `json:"service_execution_time"`
	ServiceId string `json:"service_instance_id"`
}

// EdgeInfo mirrors the edge metadata returned by /admin/edges/{systemKey}.
type EdgeInfo struct {
	Name                 string `json:"name"`
	Description          string `json:"description"`
	Token                string `json:"token"` // edge's own dev token — used for proxy auth
	PublicAddr           string `json:"public_addr"`
	LocalAddr            string `json:"local_addr"`
	LastSeenVersion      string `json:"last_seen_version"`
	LastSeenOS           string `json:"last_seen_os"`
	LastSeenArchitecture string `json:"last_seen_architecture"`
	LastConnect          int64  `json:"last_connect"`
	LastDisconnect       int64  `json:"last_disconnect"`
	IsConnected          bool   `json:"isConnected"`
}

// CollectionInfo mirrors the collection metadata returned by the platform.
type CollectionInfo struct {
	Name      string `json:"name"`
	ID        string `json:"collectionID"`
	SystemKey string `json:"appID"`
}

// CollectionData mirrors the response from a collection GET request.
type CollectionData struct {
	Data        []map[string]any `json:"DATA"`
	NextPageURL *string          `json:"NEXTPAGEURL"`
	PrevPageURL *string          `json:"PREVPAGEURL"`
}

// LogEntry mirrors a single row from the v4 code logs endpoint.
type LogEntry struct {
	Name      string `json:"name"`
	ServiceID string `json:"service_id"`
	Level     string `json:"level"`
	Log       string `json:"log"`
	Time      int64  `json:"time"` // Unix microseconds
}
