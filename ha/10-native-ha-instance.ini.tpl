NativeHALocalInstance:
  Name={{ .Name }}
  {{ if .SSLFipsRequired }}
  SSLFipsRequired={{ .SSLFipsRequired }}
  {{- end}}
