package main

import (
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := "X5vnT8KE21rrINiYaqU5rQzE"

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Password:", password)
	fmt.Println("Bcrypt Hash:", string(hash))
}
