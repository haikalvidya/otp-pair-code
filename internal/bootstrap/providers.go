package bootstrap

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now().UTC()
}

type secureOTPGenerator struct{}

func (secureOTPGenerator) Generate(context.Context) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(90000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%05d", n.Int64()+10000), nil
}
