package bingx

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// SignQuery signiert einen Query-String mit HMAC-SHA256 und liefert die Signatur als Hex.
func SignQuery(query, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	// hash.Hash.Write liefert laut Doku nie einen Fehler.
	_, _ = mac.Write([]byte(query))
	return hex.EncodeToString(mac.Sum(nil))
}
