package auth

import "golang.org/x/crypto/bcrypt"

type PasswordManager struct {
	cost int
}

func NewPasswordManager(cost int) *PasswordManager {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}

	return &PasswordManager{cost: cost}
}

func (m *PasswordManager) Hash(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), m.cost)
	if err != nil {
		return "", err
	}

	return string(hashed), nil
}

func (m *PasswordManager) Compare(hashedPassword string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
