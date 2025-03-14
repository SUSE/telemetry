package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
)

func getHash(hasher hash.Hash, target any) (hashed string, err error) {
	content, err := json.Marshal(target)
	if err != nil {
		err = fmt.Errorf("failed to json.Marshal() hash target: %w", err)
		return
	}

	_, err = hasher.Write(content)
	if err != nil {
		err = fmt.Errorf("failed to write JSON'd target to hasher: %w", err)
		return
	}

	// encode the content hash as a string
	hashed = hex.EncodeToString(hasher.Sum(nil))

	return
}

func GetMd5Hash(target any) (hashed string, err error) {
	return getHash(md5.New(), target)
}

func GetSha256Hash(target any) (hashed string, err error) {
	return getHash(sha256.New(), target)
}

func GetSha512Hash(target any) (hashed string, err error) {
	return getHash(sha512.New(), target)
}
