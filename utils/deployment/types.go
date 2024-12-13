package deployment

import (
	"time"
)

// Record represents a single deployment record
type Record struct {
	Timestamp    time.Time         `json:"timestamp"`
	VersionID    string            `json:"version_id"`
	DeployedBy   string            `json:"deployed_by"`
	Command      string            `json:"command"`
	Status       string            `json:"status"`
	Environment  string            `json:"environment"`
	Parameters   map[string]string `json:"parameters,omitempty"`
	Duration     time.Duration     `json:"duration,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
}

// History represents the deployment history file structure
type History struct {
	FormatVersion    string   `json:"format_version"`
	Environment      string   `json:"environment"`
	LatestDeployment *Record  `json:"latest_deployment,omitempty"`
	Deployments      []Record `json:"deployments"`
}

// QueryOptions represents options for querying deployments
type QueryOptions struct {
	Since      time.Time
	Before     time.Time
	Limit      int
	Status     string
	DeployedBy string
	VersionID  string
}

// Stats represents deployment statistics
type Stats struct {
	TotalDeployments  int
	SuccessfulCount   int
	FailedCount       int
	AverageDuration   time.Duration
	LastDeployment    *Record
	CommonErrors      map[string]int
	DeploymentsByUser map[string]int
}
