package search

import (
	gitea_sdk "code.gitea.io/sdk/gitea"
)

func slimUserDetail(u *gitea_sdk.User) map[string]any {
	if u == nil {
		return nil
	}
	return map[string]any{
		"id":         u.ID,
		"login":      u.UserName,
		"full_name":  u.FullName,
		"email":      u.Email,
		"avatar_url": u.AvatarURL,
		"html_url":   u.HTMLURL,
		"is_admin":   u.IsAdmin,
	}
}

func slimUserDetails(users []*gitea_sdk.User) []map[string]any {
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, slimUserDetail(u))
	}
	return out
}

func slimTeam(t *gitea_sdk.Team) map[string]any {
	if t == nil {
		return nil
	}
	return map[string]any{
		"id":          t.ID,
		"name":        t.Name,
		"description": t.Description,
		"permission":  t.Permission,
	}
}

func slimTeams(teams []*gitea_sdk.Team) []map[string]any {
	out := make([]map[string]any, 0, len(teams))
	for _, t := range teams {
		out = append(out, slimTeam(t))
	}
	return out
}

func slimRepo(r *gitea_sdk.Repository) map[string]any {
	if r == nil {
		return nil
	}
	m := map[string]any{
		"id":                r.ID,
		"full_name":         r.FullName,
		"description":       r.Description,
		"html_url":          r.HTMLURL,
		"clone_url":         r.CloneURL,
		"ssh_url":           r.SSHURL,
		"default_branch":    r.DefaultBranch,
		"private":           r.Private,
		"fork":              r.Fork,
		"archived":          r.Archived,
		"language":          r.Language,
		"stars_count":       r.Stars,
		"forks_count":       r.Forks,
		"open_issues_count": r.OpenIssues,
		"open_pr_counter":   r.OpenPulls,
		"created_at":        r.Created,
		"updated_at":        r.Updated,
	}
	if r.Owner != nil {
		m["owner"] = r.Owner.UserName
	}
	if len(r.Topics) > 0 {
		m["topics"] = r.Topics
	}
	return m
}

func slimRepos(repos []*gitea_sdk.Repository) []map[string]any {
	out := make([]map[string]any, 0, len(repos))
	for _, r := range repos {
		out = append(out, slimRepo(r))
	}
	return out
}

func userLogin(u *gitea_sdk.User) string {
	if u == nil {
		return ""
	}
	return u.UserName
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
		if i.Repository != nil {
			m["repository"] = i.Repository.FullName
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
		out = append(out, m)
	}
	return out
}
