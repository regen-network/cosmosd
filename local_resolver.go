package main

type LocalResolver struct {
	Path   string `json:"path"`
	Sha256 string `json:"sha256"`
}

func (res LocalResolver) BinaryPath() string {
	return res.Path
}

func (res LocalResolver) Sha256Hash() string {
	return res.Sha256
}

