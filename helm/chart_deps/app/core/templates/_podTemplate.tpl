{{ define "core.podtemplate" }}
{{- $ := index . 0 }}
{{- $obj := index . 1 }}
template:
  metadata:
    {{- with $obj.podAnnotations }}
    annotations:
    {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- if or $obj.selectorLabels $obj.podLabels }}
    {{- $podLabels := mustDeepCopy ($obj.podLabels | default dict) }}
    {{- $podLabels = mergeOverwrite $podLabels ($obj.selectorLabels | default dict) }}
    labels:
    {{- tpl (toYaml $podLabels) $ | nindent 6 }}
    {{- end }}
  spec:
    {{- if hasKey $obj "automountServiceAccountToken" }}
    automountServiceAccountToken: {{ $obj.automountServiceAccountToken }}
    {{- end }}
    {{- if $obj.shareProcessNamespace }}
    shareProcessNamespace: true
    {{- end }}
    {{- if ne $obj.terminationGracePeriodSeconds nil }}
    terminationGracePeriodSeconds: {{ $obj.terminationGracePeriodSeconds }}
    {{- end }}
    {{- with $obj.imagePullSecrets }}
    imagePullSecrets:
    {{- toYaml . | nindent 6 }}
    {{- end }}
    serviceAccountName: {{ tpl $obj.serviceAccountName $ }}
    {{- if $obj.restartPolicy }}
    restartPolicy: {{ $obj.restartPolicy }}
    {{- end }}
    {{- if $obj.podSecurityContext }}
    securityContext:
    {{- tpl (toYaml $obj.podSecurityContext) $ | nindent 6 }}
    {{- else }}
    securityContext: {}
    {{- end }}
    {{- if $obj.initContainers }}
    initContainers:
    {{- if kindIs "map" $obj.initContainers }}
    {{- range $name, $container := $obj.initContainers }}
    {{- $initContainer := mustDeepCopy $container }}
    {{- $initContainer = set $initContainer "name" ($container.name | default $name) }}
    {{- tpl (toYaml (list $initContainer)) $ | nindent 6 }}
    {{- end }}
    {{- else }}
    {{- tpl (toYaml $obj.initContainers) $ | nindent 4 }}
    {{- end }}
    {{- end }}
    containers:
      - name: {{ $obj.containerName }}
        {{- if $obj.securityContext }}
        securityContext:
        {{- tpl (toYaml $obj.securityContext) $ | nindent 10 }}
        {{- else }}
        securityContext: {}
        {{- end }}
        image: '{{ tpl $obj.image $ }}'
        {{- if $obj.imagePullPolicy }}
        imagePullPolicy: {{ $obj.imagePullPolicy }}
        {{- else }}
        imagePullPolicy: IfNotPresent
        {{- end }}
        {{- with $obj.command }}
        command:
        {{- tpl (toYaml .) $ | nindent 10 }}
        {{- end }}
        {{- with $obj.args }}
        args:
        {{- tpl (toYaml .) $ | nindent 10 }}
        {{- end }}
        {{- with $obj.env }}
        env:
        {{-  tpl (toYaml .) $ | nindent 10}}
        {{- end }}
        {{- with $obj.ports }}
        ports:
        {{- tpl (toYaml .) $ | nindent 10 }}
        {{- end }}
        {{- with $obj.livenessProbe }}
        livenessProbe:
        {{- tpl (toYaml .) $ | nindent 10 }}
        {{- end }}
        {{- with $obj.readinessProbe }}
        readinessProbe:
        {{-  tpl (toYaml .) $ | nindent 10 }}
        {{- end }}
        {{- with $obj.resources }}
        resources:
        {{-  tpl (toYaml .) $ | nindent 10 }}
        {{- end }}
        {{- with $obj.volumeMounts }}
        volumeMounts:
        {{- tpl (toYaml .) $ | nindent 10 }}
        {{- end }}
    {{- with $obj.sidecars }}
    {{- tpl (toYaml .) $ | nindent 6 }}
    {{- end }}
    {{- with $obj.nodeSelector }}
    nodeSelector:
    {{- tpl (toYaml .) $ | nindent 6 }}
    {{- end }}
    {{- with $obj.affinity }}
    affinity:
    {{- tpl (toYaml .) $ | nindent 6 }}
    {{- end }}
    {{- with $obj.topologySpreadConstraints }}
    topologySpreadConstraints:
    {{- tpl (toYaml .) $ | nindent 6 }}
    {{- end }}
    {{- with $obj.tolerations }}
    tolerations:
    {{- tpl (toYaml .) $ | nindent 6 }}
    {{- end }}
    {{- with $obj.priorityClassName }}
    priorityClassName: {{ tpl . $ }}
    {{- end }}
    {{- with $obj.volumes }}
    volumes: 
    {{- tpl (toYaml .) $ | nindent 6 }}
    {{- end }}
{{- end }}
