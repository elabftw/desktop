package main

import (
	"context"

	"golang.org/x/crypto/bcrypt"
)

// App struct
type App struct {
	ctx            context.Context
	index          *ProfileIndex
	passphraseHash string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	idx, err := loadProfileIndex()
	if err != nil {
		panic(err)
	}

	a.index = idx
}

func (a *App) GetProfileIndex() *ProfileIndex {
	return a.index
}

func (a *App) GetHash() string {
	return a.passphraseHash
}

func (a *App) Login(passphrase string) error {
	hash, err := hashPassphrase(passphrase)
	if err != nil {
		return err
	}

	a.passphraseHash = hash
	return nil
}

func hashPassphrase(passphrase string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword(
		[]byte(passphrase),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

func verifyPassphrase(passphrase, storedHash string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(storedHash),
		[]byte(passphrase),
	)
	return err == nil
}
