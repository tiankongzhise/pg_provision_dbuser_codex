package password

import (
	"crypto/rand"
	"errors"
	"math/big"
)

const DefaultLength = 24

var (
	lower   = []rune("abcdefghijkmnopqrstuvwxyz")
	upper   = []rune("ABCDEFGHJKLMNPQRSTUVWXYZ")
	digits  = []rune("23456789")
	symbols = []rune("!@#$%^&*()-_=+[]{}")
	all     = join(lower, upper, digits, symbols)
)

func Generate(length int) (string, error) {
	if length < 12 {
		return "", errors.New("密码长度至少需要 12 位")
	}

	runes := make([]rune, 0, length)
	for _, charset := range [][]rune{lower, upper, digits, symbols} {
		ch, err := randomRune(charset)
		if err != nil {
			return "", err
		}
		runes = append(runes, ch)
	}

	for len(runes) < length {
		ch, err := randomRune(all)
		if err != nil {
			return "", err
		}
		runes = append(runes, ch)
	}

	if err := shuffle(runes); err != nil {
		return "", err
	}
	return string(runes), nil
}

func randomRune(charset []rune) (rune, error) {
	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
	if err != nil {
		return 0, err
	}
	return charset[index.Int64()], nil
}

func shuffle(values []rune) error {
	for i := len(values) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		values[i], values[j.Int64()] = values[j.Int64()], values[i]
	}
	return nil
}

func join(groups ...[]rune) []rune {
	var out []rune
	for _, group := range groups {
		out = append(out, group...)
	}
	return out
}
