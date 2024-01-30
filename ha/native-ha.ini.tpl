NativeHALocalInstance:
  Name={{ .Name }}
  {{ if .CertificateLabel }}
  CertificateLabel={{ .CertificateLabel }}
  KeyRepository={{ .KeyRepository }}
  {{ if .CipherSpec }}
  CipherSpec={{ .CipherSpec }}
  {{- end }}
  {{ if .SSLFipsRequired }}
  SSLFipsRequired={{ .SSLFipsRequired }}
  {{- end }}
  {{- end }}
{{- range $idx, $instance := .Instances}}
NativeHAInstance:
  Name={{ $instance.Name }}
  ReplicationAddress={{ $instance.ReplicationAddress }}
{{- end}}
