package inference

import "regexp"

var (
	reUUID     = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	reEmail    = regexp.MustCompile(`(?i)^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	reDateTime = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)
	reDate     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	reURI      = regexp.MustCompile(`(?i)^https?://`)
	reIPv4     = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
)

func inferStringFormat(s string) string {
	switch {
	case reUUID.MatchString(s):
		return "uuid"
	case reDateTime.MatchString(s):
		return "date-time"
	case reDate.MatchString(s):
		return "date"
	case reEmail.MatchString(s):
		return "email"
	case reURI.MatchString(s):
		return "uri"
	case reIPv4.MatchString(s):
		return "ipv4"
	default:
		return ""
	}
}
