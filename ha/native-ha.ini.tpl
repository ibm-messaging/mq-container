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
NativeHAInstance:
  Name={{ .NativeHAInstance0_Name }}
  ReplicationAddress={{ .NativeHAInstance0_ReplicationAddress }}
NativeHAInstance:
  Name={{ .NativeHAInstance1_Name }}
  ReplicationAddress={{ .NativeHAInstance1_ReplicationAddress }}
NativeHAInstance:
  Name={{ .NativeHAInstance2_Name }}
  ReplicationAddress={{ .NativeHAInstance2_ReplicationAddress }}
