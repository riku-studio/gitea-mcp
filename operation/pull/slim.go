package pull

import (
	"fmt"
	"strings"

	gitea_sdk "code.gitea.io/sdk/gitea"
)

func bodyWithAttachments(body string, atts []*gitea_sdk.Attachment) string {
	links := make([]string, 0, len(atts))
	for _, a := range atts {
		if a == nil || a.DownloadURL == "" {
			continue
		}
		links = append(links, fmt.Sprintf("[%s](%s)", a.Name, a.DownloadURL))
	}
	if len(links) == 0 {
		return body
	}
	joined := strings.Join(links, "\n")
	if body == "" {
		return joined
	}
	return body + "\n\n" + joined
}

func userLogin(u *gitea_sdk.User) string {
	if u == nil {
		return ""
	}
	return u.UserName
}

func userLogins(users []*gitea_sdk.User) []string {
	if len(users) == 0 {
		return nil
	}
	out := make([]string, 0, len(users))
	for _, u := range users {
		if u != nil {
			out = append(out, u.UserName)
		}
	}
	return out
}

func labelNames(labels []*gitea_sdk.Label) []string {
	if len(labels) == 0 {
		return nil
	}
	out := make([]string, 0, len(labels))
	for _, l := range labels {
		if l != nil {
			out = append(out, l.Name)
		}
	}
	return out
}

func repoRef(r *gitea_sdk.Repository) map[string]any {
	if r == nil {
		return nil
	}
	return map[string]any{
		"full_name":   r.FullName,
		"description": r.Description,
	}
}

func slimPullRequest(pr *gitea_sdk.PullRequest) map[string]any {
	if pr == nil {
		return nil
	}
	m := map[string]any{
		"number":     pr.Index,
		"title":      pr.Title,
		"body":       pr.Body,
		"state":      pr.State,
		"draft":      pr.Draft,
		"merged":     pr.HasMerged,
		"mergeable":  pr.Mergeable,
		"html_url":   pr.HTMLURL,
		"user":       userLogin(pr.Poster),
		"labels":     labelNames(pr.Labels),
		"comments":   pr.Comments,
		"created_at": pr.Created,
		"updated_at": pr.Updated,
		"closed_at":  pr.Closed,
	}
	if pr.HasMerged {
		m["merged_at"] = pr.Merged
		m["merge_commit_sha"] = pr.MergedCommitID
		m["merged_by"] = userLogin(pr.MergedBy)
	}
	if pr.Head != nil {
		head := map[string]any{"ref": pr.Head.Ref, "sha": pr.Head.Sha}
		if pr.Head.Repository != nil {
			head["repo"] = repoRef(pr.Head.Repository)
		}
		m["head"] = head
	}
	if pr.Base != nil {
		base := map[string]any{"ref": pr.Base.Ref, "sha": pr.Base.Sha}
		if pr.Base.Repository != nil {
			base["repo"] = repoRef(pr.Base.Repository)
		}
		m["base"] = base
	}
	if pr.Additions != nil {
		m["additions"] = *pr.Additions
	}
	if pr.Deletions != nil {
		m["deletions"] = *pr.Deletions
	}
	if pr.ChangedFiles != nil {
		m["changed_files"] = *pr.ChangedFiles
	}
	if len(pr.Assignees) > 0 {
		m["assignees"] = userLogins(pr.Assignees)
	}
	if pr.Milestone != nil {
		m["milestone"] = pr.Milestone.Title
	}
	if pr.ReviewComments > 0 {
		m["review_comments"] = pr.ReviewComments
	}
	return m
}

func slimPullRequests(prs []*gitea_sdk.PullRequest) []map[string]any {
	out := make([]map[string]any, 0, len(prs))
	for _, pr := range prs {
		if pr == nil {
			continue
		}
		m := map[string]any{
			"number":     pr.Index,
			"title":      pr.Title,
			"state":      pr.State,
			"draft":      pr.Draft,
			"merged":     pr.HasMerged,
			"html_url":   pr.HTMLURL,
			"user":       userLogin(pr.Poster),
			"created_at": pr.Created,
			"updated_at": pr.Updated,
		}
		if pr.Head != nil {
			m["head"] = pr.Head.Ref
		}
		if pr.Base != nil {
			m["base"] = pr.Base.Ref
		}
		if len(pr.Labels) > 0 {
			m["labels"] = labelNames(pr.Labels)
		}
		out = append(out, m)
	}
	return out
}

func slimReview(r *gitea_sdk.PullReview) map[string]any {
	if r == nil {
		return nil
	}
	return map[string]any{
		"id":             r.ID,
		"state":          r.State,
		"body":           r.Body,
		"user":           userLogin(r.Reviewer),
		"comments_count": r.CodeCommentsCount,
		"submitted_at":   r.Submitted,
		"html_url":       r.HTMLURL,
		"stale":          r.Stale,
		"official":       r.Official,
		"dismissed":      r.Dismissed,
	}
}

func slimReviews(reviews []*gitea_sdk.PullReview) []map[string]any {
	out := make([]map[string]any, 0, len(reviews))
	for _, r := range reviews {
		out = append(out, slimReview(r))
	}
	return out
}

func slimReviewComment(c *gitea_sdk.PullReviewComment) map[string]any {
	if c == nil {
		return nil
	}
	return map[string]any{
		"id":           c.ID,
		"body":         c.Body,
		"path":         c.Path,
		"position":     c.LineNum,
		"old_position": c.OldLineNum,
		"diff_hunk":    c.DiffHunk,
		"user":         userLogin(c.Reviewer),
		"html_url":     c.HTMLURL,
		"created_at":   c.Created,
		"updated_at":   c.Updated,
	}
}

func slimReviewComments(comments []*gitea_sdk.PullReviewComment) []map[string]any {
	out := make([]map[string]any, 0, len(comments))
	for _, c := range comments {
		out = append(out, slimReviewComment(c))
	}
	return out
}
