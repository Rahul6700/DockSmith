package state

import (
	"encoding/json"
)

// content hashing an image
func ComputeImageDigest(img Image) (string, error) {

	img.Digest = ""

	data, err := json.Marshal(img)
	if err != nil {
		return "", err
	}

	return HashBytes(data), nil
}
