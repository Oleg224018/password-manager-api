package data

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/scrypt"
)

const (
	DATA_FILE = "passwords.json.encrypted"
	SALT_SIZE = 32
)

func deriveKey(password string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
}

func encrypt(plaintext []byte, password string) ([]byte, error) {
	salt := make([]byte, SALT_SIZE)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	key, err := deriveKey(password, salt)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return append(salt, ciphertext...), nil
}

func decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < SALT_SIZE {
		return nil, fmt.Errorf("повреждённый файл")
	}

	salt := data[:SALT_SIZE]
	ciphertext := data[SALT_SIZE:]

	key, err := deriveKey(password, salt)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("повреждённый файл")
	}

	nonce, encrypted := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("неверный мастер-пароль")
	}

	return plaintext, nil
}

func SaveEncrypted(data AppData, password string) error {
	plaintext, err := json.Marshal(data)
	if err != nil {
		return err
	}

	encrypted, err := encrypt(plaintext, password)
	if err != nil {
		return err
	}

	return os.WriteFile(DATA_FILE, encrypted, 0600)
}

func LoadEncrypted(password string) (AppData, error) {
	var data AppData

	encrypted, err := os.ReadFile(DATA_FILE)
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return data, err
	}

	plaintext, err := decrypt(encrypted, password)
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(plaintext, &data)
	return data, err
}

func (a *AppData) GetCategoryID(name string) string {
	for _, c := range a.Categories {
		if c.Name == name {
			return c.ID
		}
	}
	id := "c" + strconv.FormatInt(time.Now().UnixNano()%100000, 10)
	a.Categories = append(a.Categories, Category{ID: id, Name: name})
	return id
}

func (a *AppData) GetCategoryName(id string) string {
	for _, c := range a.Categories {
		if c.ID == id {
			return c.Name
		}
	}
	return "—"
}

func (a *AppData) FindEntryIndex(id string) int {
	for i, e := range a.Entries {
		if e.ID == id {
			return i
		}
	}
	return -1
}
