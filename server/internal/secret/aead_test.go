package secret

import "testing"

func TestAEADEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	cipher, err := NewAEAD("integration-secret")
	if err != nil {
		t.Fatalf("NewAEAD() error = %v", err)
	}

	ciphertext, err := cipher.Encrypt([]byte("REVEL_SESSION=test"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if len(ciphertext) == 0 {
		t.Fatal("Encrypt() returned empty ciphertext")
	}

	plaintext, err := cipher.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if string(plaintext) != "REVEL_SESSION=test" {
		t.Fatalf("Decrypt() plaintext = %q, want %q", string(plaintext), "REVEL_SESSION=test")
	}
}
