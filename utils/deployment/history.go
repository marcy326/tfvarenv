package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"tfvarenv/config"
	"tfvarenv/utils/aws"
)

const HistoryFormatVersion = "1.0"

const (
	StatusSuccess = "success"
	StatusFailure = "failure"
)

const (
	StatusActive    = "active"
	StatusDestroyed = "destroyed"
)

const (
	CommandApply   = "apply"
	CommandPlan    = "plan"
	CommandDestroy = "destroy"
)

type Manager interface {
	AddRecord(ctx context.Context, record *Record) error
	GetHistory(ctx context.Context) (*History, error)
	GetLatestDeployment(ctx context.Context) (*Record, error)
	GetStats(ctx context.Context) (*Stats, error)
	QueryDeployments(ctx context.Context, options QueryOptions) ([]Record, error)
	MarkAsDestroyed(ctx context.Context) error
}

type manager struct {
	awsClient aws.Client
	env       *config.Environment
}

func NewManager(awsClient aws.Client, env *config.Environment) Manager {
	return &manager{
		awsClient: awsClient,
		env:       env,
	}
}

func (m *manager) AddRecord(ctx context.Context, record *Record) error {
	history, err := m.GetHistory(ctx)
	if err != nil {
		return fmt.Errorf("failed to get deployment history: %w", err)
	}

	// Add new record
	history.Deployments = append(history.Deployments, *record)

	history.LatestDeployment = &LatestInfo{
		Deployment:   record,
		Status:       StatusActive,
		ModifiedTime: time.Now(),
	}

	// Sort deployments by timestamp (newest first)
	sort.Slice(history.Deployments, func(i, j int) bool {
		return history.Deployments[i].Timestamp.After(history.Deployments[j].Timestamp)
	})

	// Save updated history
	if err := m.saveHistory(ctx, history); err != nil {
		return fmt.Errorf("failed to save deployment history: %w", err)
	}

	return nil
}

// MarkAsDestroyed は環境の状態をdestroyedに設定
func (m *manager) MarkAsDestroyed(ctx context.Context) error {
	history, err := m.GetHistory(ctx)
	if err != nil {
		return fmt.Errorf("failed to get deployment history: %w", err)
	}

	if history.LatestDeployment == nil {
		history.LatestDeployment = &LatestInfo{}
	}
	history.LatestDeployment.Status = StatusDestroyed
	history.LatestDeployment.ModifiedTime = time.Now()

	if err := m.saveHistory(ctx, history); err != nil {
		return fmt.Errorf("failed to save deployment history: %w", err)
	}

	return nil
}

func (m *manager) GetHistory(ctx context.Context) (*History, error) {
	input := &aws.DownloadInput{
		Bucket: m.env.S3.Bucket,
		Key:    m.env.GetDeploymentHistoryKey(),
	}

	output, err := m.awsClient.DownloadFile(ctx, input)
	if err != nil {
		// 履歴ファイルが存在しない場合は新規作成
		return &History{
			FormatVersion: HistoryFormatVersion,
			Environment:   m.env.Name,
			Deployments:   make([]Record, 0),
		}, nil
	}

	var history History
	if err := json.Unmarshal(output.Content, &history); err != nil {
		return nil, fmt.Errorf("failed to decode deployment history: %w", err)
	}

	return &history, nil
}

func (m *manager) GetLatestDeployment(ctx context.Context) (*Record, error) {
	history, err := m.GetHistory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment history: %w", err)
	}

	if history.LatestDeployment != nil && history.LatestDeployment.Deployment != nil {
		return history.LatestDeployment.Deployment, nil
	}

	if len(history.Deployments) > 0 {
		return &history.Deployments[0], nil
	}

	return nil, nil
}

func (m *manager) GetStats(ctx context.Context) (*Stats, error) {
	history, err := m.GetHistory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment history: %w", err)
	}

	stats := &Stats{
		TotalDeployments:  len(history.Deployments),
		CommonErrors:      make(map[string]int),
		DeploymentsByUser: make(map[string]int),
	}

	var totalDuration time.Duration
	for _, d := range history.Deployments {
		if d.Status == StatusSuccess {
			stats.SuccessfulCount++
		} else {
			stats.FailedCount++
			if d.ErrorMessage != "" {
				stats.CommonErrors[d.ErrorMessage]++
			}
		}

		stats.DeploymentsByUser[d.DeployedBy]++
		if d.Duration > 0 {
			totalDuration += d.Duration
		}
	}

	if stats.TotalDeployments > 0 {
		stats.AverageDuration = totalDuration / time.Duration(stats.TotalDeployments)
	}

	if len(history.Deployments) > 0 {
		stats.LastDeployment = &history.Deployments[0]
	}

	return stats, nil
}

func (m *manager) QueryDeployments(ctx context.Context, options QueryOptions) ([]Record, error) {
	history, err := m.GetHistory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment history: %w", err)
	}

	var filtered []Record
	for _, d := range history.Deployments {
		if !options.Since.IsZero() && d.Timestamp.Before(options.Since) {
			continue
		}
		if !options.Before.IsZero() && d.Timestamp.After(options.Before) {
			continue
		}
		if options.Status != "" && d.Status != options.Status {
			continue
		}
		if options.DeployedBy != "" && d.DeployedBy != options.DeployedBy {
			continue
		}
		if options.VersionID != "" && d.VersionID != options.VersionID {
			continue
		}
		filtered = append(filtered, d)
	}

	if options.Limit > 0 && len(filtered) > options.Limit {
		filtered = filtered[:options.Limit]
	}

	return filtered, nil
}

func (m *manager) saveHistory(ctx context.Context, history *History) error {
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal deployment history: %w", err)
	}

	input := &aws.UploadInput{
		Bucket:      m.env.S3.Bucket,
		Key:         m.env.GetDeploymentHistoryKey(),
		Content:     data,
		ContentType: "application/json",
	}

	if _, err := m.awsClient.UploadFile(ctx, input); err != nil {
		return fmt.Errorf("failed to save deployment history: %w", err)
	}

	return nil
}
