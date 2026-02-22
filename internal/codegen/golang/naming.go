package golang

import "strings"

var commonInitialisms = map[string]bool{
	"ACL": true, "API": true, "ASCII": true, "CPU": true, "CSS": true,
	"DNS": true, "EOF": true, "GUID": true, "HTML": true, "HTTP": true,
	"HTTPS": true, "ID": true, "IP": true, "JSON": true, "LHS": true,
	"QPS": true, "RAM": true, "RHS": true, "RPC": true, "SLA": true,
	"SMTP": true, "SQL": true, "SSH": true, "SSL": true, "TCP": true,
	"TLS": true, "TTL": true, "UDP": true, "UI": true, "UID": true,
	"UUID": true, "URI": true, "URL": true, "UTF8": true, "VM": true,
	"XML": true, "XMPP": true, "XSRF": true, "XSS": true,
}

func isCommonInitialism(s string) bool {
	return commonInitialisms[s]
}

func splitWords(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
}

func toPascalCase(s string) string {
	words := splitWords(s)
	var result strings.Builder

	for _, word := range words {
		if len(word) == 0 {
			continue
		}
		upper := strings.ToUpper(word)
		if isCommonInitialism(upper) {
			result.WriteString(upper)
		} else {
			result.WriteString(strings.ToUpper(string(word[0])))
			result.WriteString(strings.ToLower(word[1:]))
		}
	}

	return result.String()
}

func toCamelCase(s string) string {
	words := splitWords(s)
	var result strings.Builder

	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		upper := strings.ToUpper(word)
		if i == 0 {
			if isCommonInitialism(upper) {
				result.WriteString(strings.ToLower(upper))
			} else {
				result.WriteString(strings.ToLower(word))
			}
		} else {
			if isCommonInitialism(upper) {
				result.WriteString(upper)
			} else {
				result.WriteString(strings.ToUpper(string(word[0])))
				result.WriteString(strings.ToLower(word[1:]))
			}
		}
	}

	return result.String()
}

func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	runes := []rune(s)

	for i := range len(runes) {
		r := runes[i]

		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				prevLower := runes[i-1] >= 'a' && runes[i-1] <= 'z'
				nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'

				if prevLower || nextLower {
					result.WriteRune('_')
				}
			}
			result.WriteRune(r + 32)
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}
