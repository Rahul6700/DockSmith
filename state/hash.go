package state

import (
	"crypto/sha256"
	"fmt"
)

func HashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum) // returns in the format of ""
}
