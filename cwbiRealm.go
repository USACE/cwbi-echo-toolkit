package cwbiechotoolkit

import (
	"slices"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
)

func GetRsaPublicKey(publicKey string) (*rsa.PublicKey, error) {
	return jwt.ParseRSAPublicKeyFromPEM([]byte("-----BEGIN PUBLIC KEY-----\n" + publicKey + "\n-----END PUBLIC KEY-----"))
}

func GetPublicKeyFromCwbiRealm(url string) (string, error) {
	// Make the HTTP GET request
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	// Read the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	// Parse JSON
	var data map[string]any
	err = json.Unmarshal(body, &data)

	return data["public_key"].(string), err
}

func GetRsaPublicKeyFromCwbiRealm(url string) (*rsa.PublicKey, error) {
	publicKey, err := GetPublicKeyFromCwbiRealm(url)
	if err != nil {
		return nil, err
	}

	return GetRsaPublicKey(publicKey)
}

func StringArrayMatch(arr1 []string, arr2 []string) bool {
	for _, v1 := range arr1 {
		if slices.Contains(arr2, v1) {
				return true
			}
	}
	return false
}
