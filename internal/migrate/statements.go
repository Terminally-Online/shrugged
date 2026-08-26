package migrate

import "strings"

// Statements splits SQL into its top-level statements. Postgres runs a
// multi-statement simple-protocol message as one implicit transaction, which
// is exactly what a no-transaction migration must avoid, so its statements
// are sent one at a time. The split respects single- and double-quoted
// strings, dollar-quoted bodies, and both comment forms; a trailing fragment
// without a semicolon is a statement too, and empty fragments are dropped.
func Statements(sql string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" && !onlyComments(s) {
			out = append(out, s)
		}
		cur.Reset()
	}
	i := 0
	for i < len(sql) {
		c := sql[i]
		switch {
		case c == '-' && i+1 < len(sql) && sql[i+1] == '-':
			end := strings.IndexByte(sql[i:], '\n')
			if end < 0 {
				cur.WriteString(sql[i:])
				i = len(sql)
			} else {
				cur.WriteString(sql[i : i+end+1])
				i += end + 1
			}
		case c == '/' && i+1 < len(sql) && sql[i+1] == '*':
			end := strings.Index(sql[i+2:], "*/")
			if end < 0 {
				cur.WriteString(sql[i:])
				i = len(sql)
			} else {
				cur.WriteString(sql[i : i+end+4])
				i += end + 4
			}
		case c == '\'' || c == '"':
			j := i + 1
			for j < len(sql) {
				if sql[j] == c {
					if j+1 < len(sql) && sql[j+1] == c {
						j += 2
						continue
					}
					break
				}
				j++
			}
			if j < len(sql) {
				j++
			}
			cur.WriteString(sql[i:j])
			i = j
		case c == '$':
			tag := dollarTag(sql[i:])
			if tag == "" {
				cur.WriteByte(c)
				i++
				continue
			}
			end := strings.Index(sql[i+len(tag):], tag)
			if end < 0 {
				cur.WriteString(sql[i:])
				i = len(sql)
			} else {
				cur.WriteString(sql[i : i+len(tag)+end+len(tag)])
				i += len(tag) + end + len(tag)
			}
		case c == ';':
			flush()
			i++
		default:
			cur.WriteByte(c)
			i++
		}
	}
	flush()
	return out
}

func dollarTag(s string) string {
	if len(s) < 2 || s[0] != '$' {
		return ""
	}
	j := 1
	for j < len(s) && (s[j] == '_' || (s[j] >= 'a' && s[j] <= 'z') || (s[j] >= 'A' && s[j] <= 'Z') || (j > 1 && s[j] >= '0' && s[j] <= '9')) {
		j++
	}
	if j < len(s) && s[j] == '$' {
		return s[:j+1]
	}
	return ""
}

func onlyComments(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if t != "" && !strings.HasPrefix(t, "--") {
			return false
		}
	}
	return true
}
