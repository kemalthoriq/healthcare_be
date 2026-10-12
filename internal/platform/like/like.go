package like

import "strings"

var escaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func Contains(s string) string {
	if s == "" {
		return ""
	}
	return "%" + escaper.Replace(s) + "%"
}
