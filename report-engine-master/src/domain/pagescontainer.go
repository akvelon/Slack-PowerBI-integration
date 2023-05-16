package domain

import (
	"encoding/json"
	"io"
)

// PagesContainer holds Page set.
type PagesContainer struct {
	Value []*Page `json:"value"`
}

// DeserializePagesContainer unmarshals PagesContainer from io.Reader.
func DeserializePagesContainer(b io.Reader) (*PagesContainer, error) {
	out := PagesContainer{}
	d := json.NewDecoder(b)
	if err := d.Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}
