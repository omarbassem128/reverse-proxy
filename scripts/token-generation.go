package main

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"log"
	"os"
	"time"
)

func main() {
	godotenv.Load(".env")
	secretKey := os.Getenv("JWT_SECRET")
	slicedSecretKey := []byte(secretKey)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user_12345",                          
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})

	signature, err := token.SignedString(slicedSecretKey)

	if err != nil {
		log.Fatalf("error signing token: %s", err)
	}

	fmt.Println(signature)

}
