package cwbiechotoolkit

import (
	"slices"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
)

// GetRsaPublicKey jwt ParseRSAPublicKeyFromPEM returning rsa.PublicKey
//
// Parameter:
// publicKey is the public as a string
func GetRsaPublicKey(publicKey string) (*rsa.PublicKey, error) {
	return jwt.ParseRSAPublicKeyFromPEM([]byte("-----BEGIN PUBLIC KEY-----\n" + publicKey + "\n-----END PUBLIC KEY-----"))
}

// GetPublicKeyFromCwbiRealm gets the public_key from the KeyCloak CWBI Realm
// assuming the URL is one of the correct ./auth/realms/cwbi
//
// Parameter:
// url is the URL as a string
//
// Return:
// string, error
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

// GetRsaPublicKeyFromCwbiRealm gets the public_key from the KeyCloak CWBI Realm
// assuming the URL is one of the correct ./auth/realms/cwbi
//
// Parameter:
// url is the URL as a string
//
// Return:
// *rsa.PublicKey, error
func GetRsaPublicKeyFromCwbiRealm(url string) (*rsa.PublicKey, error) {
	publicKey, err := GetPublicKeyFromCwbiRealm(url)
	if err != nil {
		return nil, err
	}

	return GetRsaPublicKey(publicKey)
}

// StringArrayMatch checks string arrays for matching values
//
// Return:
// true if array1 has value in array2 else false
func StringArrayMatch(arr1 []string, arr2 []string) bool {
	for _, v1 := range arr1 {
		if slices.Contains(arr2, v1) {
				return true
			}
	}
	return false
}
