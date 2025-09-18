package lib

import "strings"

type TypeMapping struct {
	PrefixToType map[string]string
	TypeToID     map[string]string
}

func NewTypeMapping() *TypeMapping {
	return &TypeMapping{
		PrefixToType: map[string]string{
			"feat":      "Feature",
			"bug":       "Bug",
			"docs":      "Docs",
			"blog":      "Blog",
			"interrupt": "Interrupt",
			"spike":     "Spike",
			"chore":     "Chore",
		},
		TypeToID: make(map[string]string),
	}
}

func (tm *TypeMapping) GetTypeFromTitle(title string) (string, bool) {
	colonIndex := strings.Index(title, ":")
	if colonIndex == -1 {
		return "", false
	}

	prefix := strings.TrimSpace(title[:colonIndex])

	if parenIndex := strings.Index(prefix, "("); parenIndex != -1 {
		prefix = strings.TrimSpace(prefix[:parenIndex])
	}

	prefix = strings.ToLower(prefix)

	if typeName, exists := tm.PrefixToType[prefix]; exists {
		return typeName, true
	}

	return "", false
}

func (tm *TypeMapping) SetTypeID(typeName, fieldID string) {
	tm.TypeToID[typeName] = fieldID
}

func (tm *TypeMapping) GetTypeID(typeName string) (string, bool) {
	fieldID, exists := tm.TypeToID[typeName]
	return fieldID, exists
}
