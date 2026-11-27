package lib

import "strings"

// SkipList is the repositories whose issues and pull requests stay off the project.
type SkipList map[string]bool

// ParseSkipList reads a comma-separated list of owner/name repositories.
func ParseSkipList(s string) SkipList {
	skip := SkipList{}
	for _, repo := range strings.Split(s, ",") {
		if repo = strings.TrimSpace(repo); repo != "" {
			skip[strings.ToLower(repo)] = true
		}
	}
	return skip
}

// Skips reports whether repo, as owner/name, is on the list.
func (s SkipList) Skips(repo string) bool { return s[strings.ToLower(repo)] }
