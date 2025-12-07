package pulumi

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

// GetStackHistory returns the history of updates for a stack
func GetStackHistory(ctx context.Context, workDir, stackName string, pageSize, page int, env map[string]string) ([]UpdateSummary, error) {
	resolvedStackName, err := resolveStackName(ctx, workDir, stackName, env)
	if err != nil {
		return nil, err
	}

	wsOpts := []auto.LocalWorkspaceOption{auto.WorkDir(workDir)}
	if len(env) > 0 {
		wsOpts = append(wsOpts, auto.EnvVars(env))
	}

	stack, err := auto.SelectStackLocalSource(ctx, resolvedStackName, workDir, wsOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to select stack: %w", err)
	}

	history, err := stack.History(ctx, pageSize, page)
	if err != nil {
		return nil, fmt.Errorf("failed to get stack history: %w", err)
	}

	result := make([]UpdateSummary, 0, len(history))
	for _, h := range history {
		summary := UpdateSummary{
			Version:   h.Version,
			Kind:      h.Kind,
			StartTime: h.StartTime,
			Message:   h.Message,
			Result:    h.Result,
		}
		if h.EndTime != nil {
			summary.EndTime = *h.EndTime
		}
		if h.ResourceChanges != nil {
			summary.ResourceChanges = make(map[string]int)
			for k, v := range *h.ResourceChanges {
				summary.ResourceChanges[k] = v
			}
		}
		// Extract user info from environment
		if h.Environment != nil {
			if author, ok := h.Environment["git.author"]; ok && author != "" {
				summary.User = author
			} else if committer, ok := h.Environment["git.committer"]; ok && committer != "" {
				summary.User = committer
			}
			if email, ok := h.Environment["git.author.email"]; ok && email != "" {
				summary.UserEmail = email
			} else if email, ok := h.Environment["git.committer.email"]; ok && email != "" {
				summary.UserEmail = email
			}
		}
		result = append(result, summary)
	}

	return result, nil
}
