NativeHALocalInstance:
  {{ if .ShouldConfigureTLS }}
  {{ if .CipherSpec }}
  CipherSpec={{ .CipherSpec }}
  {{- end }}
  {{ if .Group.Local.Name }}
  GroupName={{ .Group.Local.Name }}
  {{- end}}
  {{ if .Group.CipherSpec }}
  GroupCipherSpec={{ .Group.CipherSpec }}
  {{- end }}
  {{ if .Group.Local.Role }}
  GroupRole={{ .Group.Local.Role }}
  {{- end}}
  {{ if .Group.Local.Address }}
  GroupLocalAddress={{ .Group.Local.Address }}
  {{- end}}
  {{- end }}{{/* end if .ShouldConfigureTLS */}}
{{- range $idx, $instance := .Instances}}
NativeHAInstance:
  Name={{ $instance.Name }}
  ReplicationAddress={{ $instance.ReplicationAddress }}
{{- end}}
{{ if .Group.Recovery.Name }}
NativeHARecoveryGroup:
  GroupName={{ .Group.Recovery.Name }}
  Enabled={{ .Group.Recovery.Enabled }}
  ReplicationAddress={{ .Group.Recovery.Address }}
{{- end }}
