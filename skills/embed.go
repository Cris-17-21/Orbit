// Package skills provides Orbit's built-in project skills.
package skills

import _ "embed"

//go:embed orbit/SKILL.md
var orbitSkill []byte

// Orbit returns a copy of Orbit's canonical orchestration skill.
func Orbit() []byte {
	return append([]byte(nil), orbitSkill...)
}
