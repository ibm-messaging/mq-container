NativeHALocalInstance:
  {{ if .CertificateLabel }}
  CertificateLabel={{ .CertificateLabel }}
  {{- end }}
  {{ if .Group.CertificateLabel }}
  GroupCertificateLabel={{ .Group.CertificateLabel}}
  {{- end }}
  KeyRepository={{ .KeyRepository }}
