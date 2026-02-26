package identity

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

// holonTemplate generates the complete HOLON.md file content.
var holonTemplate = `---
# Holon Identity v1
uuid: {{ .UUID | quote }}
given_name: {{ .GivenName | quote }}
family_name: {{ .FamilyName | quote }}
motto: {{ .Motto | quote }}
composer: {{ .Composer | quote }}
clade: {{ .Clade | quote }}
status: {{ .Status }}
born: {{ .Born | quote }}

# Lineage
parents: [{{ joinQuoted .Parents }}]
reproduction: {{ .Reproduction | quote }}

# Optional
aliases: [{{ joinQuoted .Aliases }}]

# Metadata
generated_by: {{ .GeneratedBy | quote }}
lang: {{ .Lang | quote }}
proto_status: {{ .ProtoStatus }}
---

# {{ .GivenName }} {{ .FamilyName }}

> *"{{ .Motto }}"*

## Description

<Describe what this holon does.>

## Introspection Notes

<Any assumptions or ambiguities noted during creation.>
`

var tmplFuncs = template.FuncMap{
	"quote": func(s string) string {
		return fmt.Sprintf("%q", s)
	},
	"joinQuoted": func(ss []string) string {
		quoted := make([]string, len(ss))
		for i, s := range ss {
			quoted[i] = fmt.Sprintf("%q", s)
		}
		return strings.Join(quoted, ", ")
	},
}

// WriteHolonMD renders an Identity to a HOLON.md file at the given path.
func WriteHolonMD(id Identity, path string) error {
	tmpl, err := template.New("holon").Funcs(tmplFuncs).Parse(holonTemplate)
	if err != nil {
		return fmt.Errorf("template error: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", path, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, id); err != nil {
		return fmt.Errorf("template execution error: %w", err)
	}
	return nil
}
