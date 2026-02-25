package golang

import "strings"

var commonInitialisms = map[string]bool{
	"ACL": true, "API": true, "ASCII": true, "AUTH": true, "AWS": true,
	"CDN": true, "CIDR": true, "CLI": true, "CORS": true, "CPU": true,
	"CSS": true, "CSV": true, "DB": true, "DHCP": true, "DNS": true,
	"DOM": true, "DSN": true, "EOF": true, "FTP": true, "GCP": true,
	"GCS": true, "GPU": true, "GRPC": true, "GUID": true, "HMAC": true,
	"HTML": true, "HTTP": true, "HTTPS": true, "ID": true, "IMAP": true,
	"IOT": true, "IP": true, "JPEG": true, "JSON": true, "JWT": true,
	"KMS": true, "KV": true, "LDAP": true, "LHS": true, "MAC": true,
	"MD5": true, "MFA": true, "MQTT": true, "NAT": true, "NFS": true,
	"OAUTH": true, "OIDC": true, "OS": true, "PDF": true, "PEM": true,
	"PGP": true, "PNG": true, "QPS": true, "QUIC": true, "RAM": true,
	"RBAC": true, "REST": true, "RHS": true, "RPC": true, "RSA": true,
	"S3": true, "SAML": true, "SDK": true, "SFTP": true, "SHA": true,
	"SLA": true, "SMTP": true, "SNMP": true, "SNS": true, "SQL": true,
	"SQS": true, "SSD": true, "SSH": true, "SSL": true, "SSO": true,
	"SVG": true, "TCP": true, "TLS": true, "TOML": true, "TSV": true,
	"TTL": true, "UDP": true, "UI": true, "UID": true, "URI": true,
	"URL": true, "UTC": true, "UTF8": true, "UUID": true, "VM": true,
	"VPC": true, "VPN": true, "WASM": true, "WSS": true, "XML": true,
	"XMPP": true, "XSRF": true, "XSS": true, "YAML": true,
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
