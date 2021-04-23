package types

type TSIGKey struct {
	Algorithm string `json:"algorithm"`
	Name      string `json:"name"`
	Secret    string `json:"secret"`
}
