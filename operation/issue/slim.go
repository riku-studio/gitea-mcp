package issue

import (
	gitea_sdk "code.gitea.io/sdk/gitea"
)

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

func slimIssue(i *gitea_sdk.Issue) map[string]any {
	if i == nil {
		return nil
	}
	m := map[string]any{
		"number":     i.Index,
		"title":      i.Title,
		"body":       i.Body,
		"state":      i.State,
		"html_url":   i.HTMLURL,
		"user":       userLogin(i.Poster),
		"labels":     labelNames(i.Labels),
		"comments":   i.Comments,
		"created_at": i.Created,
		"updated_at": i.Updated,
		"closed_at":  i.Closed,
	}
	if len(i.Assignees) > 0 {
		m["assignees"] = userLogins(i.Assignees)
	}
	if i.Milestone != nil {
		m["milestone"] = map[string]any{
			"id":    i.Milestone.ID,
			"title": i.Milestone.Title,
		}
	}
	if i.Ref != "" {
		m["ref"] = i.Ref
	}
	if i.Deadline != nil {
		m["deadline"] = i.Deadline
	}
	if i.PullRequest != nil {
		m["is_pull"] = true
	}
	return m
}

func slimIssues(issues []*gitea_sdk.Issue) []map[string]any {
	out := make([]map[string]any, 0, len(issues))
	for _, i := range issues {
		if i == nil {
			continue
		}
		m := map[string]any{
			"number":     i.Index,
			"title":      i.Title,
			"state":      i.State,
			"html_url":   i.HTMLURL,
			"user":       userLogin(i.Poster),
			"comments":   i.Comments,
			"created_at": i.Created,
			"updated_at": i.Updated,
		}
		if len(i.Labels) > 0 {
			m["labels"] = labelNames(i.Labels)
		}
		if i.Ref != "" {
			m["ref"] = i.Ref
		}
		if i.Deadline != nil {
			m["deadline"] = i.Deadline
		}
		out = append(out, m)
	}
	return out
}

func slimComment(c *gitea_sdk.Comment) map[string]any {
	if c == nil {
		return nil
	}
	return map[string]any{
		"id":         c.ID,
		"body":       c.Body,
		"user":       userLogin(c.Poster),
		"html_url":   c.HTMLURL,
		"created_at": c.Created,
		"updated_at": c.Updated,
	}
}

func slimComments(comments []*gitea_sdk.Comment) []map[string]any {
	out := make([]map[string]any, 0, len(comments))
	for _, c := range comments {
		out = append(out, slimComment(c))
	}
	return out
}

func slimLabels(labels []*gitea_sdk.Label) []map[string]any {
	out := make([]map[string]any, 0, len(labels))
	for _, l := range labels {
		if l == nil {
			continue
		}
		out = append(out, map[string]any{
			"id":          l.ID,
			"name":        l.Name,
			"color":       l.Color,
			"description": l.Description,
			"exclusive":   l.Exclusive,
		})
	}
	return out
}
