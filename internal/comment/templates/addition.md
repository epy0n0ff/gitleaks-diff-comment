🔒 **Gitleaks Exclusion Added**

`{{ .FilePattern }}` {{ if .HasLineNumber }}(line {{ .LineNumber }}) {{ end }}will be excluded from secret scanning.

[View file]({{ .FileLink }})

{{ if .IsPattern }}
⚠️ **Security Note**: This wildcard pattern will match multiple files. All matching files will be excluded from gitleaks scanning.
{{ else }}
⚠️ **Security Note**: This file will no longer be scanned by gitleaks. Ensure this exclusion is intentional and necessary.
{{ end }}