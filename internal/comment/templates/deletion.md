✅ **Gitleaks Exclusion Removed**

`{{ .FilePattern }}` {{ if .HasLineNumber }}(line {{ .LineNumber }}) {{ end }}will now be scanned by gitleaks.

[View file]({{ .FileLink }})

{{ if .IsPattern }}
✅ All files matching this pattern will now be included in security scanning.
{{ else }}
✅ This file will now be included in gitleaks secret scanning.
{{ end }}