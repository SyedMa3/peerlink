package handshake

import (
	"testing"
)

func TestEncrypt(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	data := []byte("Hello, World!")
	t.Log("length before encryption", len(data))
	encrypted, err := Encrypt(key, data)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	t.Log("length after encryption", len(encrypted))

	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(data) {
		t.Fatalf("Decrypted data does not match original data")
	}
}
