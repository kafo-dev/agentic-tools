// Package skill renders a command's skill document, the text the command
// prints under -skill.
package skill

import "strings"

// placeholder marks where the usage line lands in a skill template.
const placeholder = "{{usage}}"

// Render returns template with the string usage returns in place of every
// {{usage}}. Bytes outside the placeholder pass through untouched.
func Render(usage func() string, template string) string {
	return strings.ReplaceAll(template, placeholder, usage())
}
