package auth

import "testing"

func TestPasswordManagerHashesAndCompares(t *testing.T) {
	t.Parallel()

	manager := NewPasswordManager(4)

	hashed, err := manager.Hash("secret-pass")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if hashed == "secret-pass" {
		t.Fatal("Hash() should not return the original password")
	}

	if err := manager.Compare(hashed, "secret-pass"); err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
}
