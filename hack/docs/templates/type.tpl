{{ define "type" }}
<h3 id="{{ anchorIDForType . }}">{{ .Name.Name }}{{ if eq .Kind "Alias" }} (<code>{{ .Underlying }}</code> alias){{ end }}</h3>
{{ with (typeReferences .) }}
<p><em>Appears on:</em>
{{ range $index, $type := . }}{{ if $index }}, {{ end }}<a href="{{ linkForType $type }}">{{ typeDisplayName $type }}</a>{{ end }}
</p>
{{ end }}
<div>{{ safe (renderComments .CommentLines) }}</div>
{{ with (constantsOfType .) }}
<table>
<thead><tr><th>Value</th><th>Description</th></tr></thead>
<tbody>
{{ range . }}
<tr><td><p>{{ typeDisplayName . }}</p></td><td>{{ safe (renderComments .CommentLines) }}</td></tr>
{{ end }}
</tbody>
</table>
{{ end }}
{{ if .Members }}
<table>
<thead><tr><th>Field</th><th>Description</th></tr></thead>
<tbody>
{{ if isExportedType . }}
<tr><td><code>apiVersion</code><br/>string</td><td><code>{{ apiGroup . }}</code></td></tr>
<tr><td><code>kind</code><br/>string</td><td><code>{{ .Name.Name }}</code></td></tr>
{{ end }}
{{ template "members" . }}
</tbody>
</table>
{{ end }}
{{ end }}
