package controllers

import "strings"

func parseTags(query map[string][]string) []string {
	var tags []string
	for _, value := range query["tags"] {
		for _, tag := range strings.Split(value, ",") {
			if strings.TrimSpace(tag) != "" {
				tags = append(tags, tag)
			}
		}
	}
	if len(tags) > 0 {
		return tags
	}
	return query["tag"]
}
