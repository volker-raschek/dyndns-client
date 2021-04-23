package types

type Zone struct {
	DNSServer   string `json:"dns-server"`
	Name        string `json:"name"`
	TSIGKeyName string `json:"tsig-key"`
}
