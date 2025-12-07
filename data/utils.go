package data

import (
	"math/rand"
	"strconv"
	"time"
)

var globalRand *rand.Rand

func init() {
	globalRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func NewID() string {
	return "p" + strconv.FormatInt(time.Now().UnixNano()%1000000, 10)
}

func GeneratePassword(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[globalRand.Intn(len(chars))]
	}
	return string(b)
}
