{{ define "members" }}
{{ range .Members }}
{{ if not (hiddenMember .) }}
{{ if fieldEmbedded . }}
{{ template "members" .Type }}
{{ else }}
<tr>
<td><code>{{ fieldName . }}</code><br/>
<em>{{ if linkForType .Type }}<a href="{{ linkForType .Type }}">{{ typeDisplayName .Type }}</a>{{ else }}{{ typeDisplayName .Type }}{{ end }}</em>
</td>
<td>
{{ if isOptionalMember . }}<em>(Optional)</em>{{ end }}
{{ safe (renderComments .CommentLines) }}
{{ if eq .Type.Name.Name "ObjectMeta" }}
Refer to the Kubernetes API documentation for the fields of <code>metadata</code>.
{{ end }}
</td>
</tr>
{{ end }}
{{ end }}
{{ end }}
{{ end }}
