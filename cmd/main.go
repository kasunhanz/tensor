package main

import (
	"github.com/gamunu/tensor/util"
	"os"
	"fmt"
	"log"
	"bufio"
	"github.com/gamunu/tensor/models"
	database "github.com/gamunu/tensor/db"
	"strings"
	"gopkg.in/mgo.v2/bson"
	"golang.org/x/crypto/bcrypt"
	"time"
)

func main() {
	if util.InteractiveSetup {
		os.Exit(doSetup())
	}
}

func doSetup() int {
	fmt.Println("Checking database connectivity.. Please be patient.")

	if err := database.Connect(); err != nil {
		log.Fatal("\n Cannot connect to database!\n" + err.Error())
	}

	stdin := bufio.NewReader(os.Stdin)

	var user models.User
	user.Username = readNewline("\n\n > Username: ", stdin)
	user.Username = strings.ToLower(user.Username)
	user.Email = readNewline(" > Email: ", stdin)
	user.Email = strings.ToLower(user.Email)

	var existingUser models.User

	userc := database.MongoDb.C("users")
	err := userc.Find(bson.M{"email": user.Email, "username": user.Username}).One(&existingUser)

	if err == nil {
		// user already exists
		fmt.Printf("\n Welcome back, %v! (a user with this username/email is already set up..)\n\n", existingUser.Name)
	} else {
		user.Name = readNewline(" > Your name: ", stdin)
		user.Password = readNewline(" > Password: ", stdin)
		pwdHash, _ := bcrypt.GenerateFromPassword([]byte(user.Password), 11)

		newUser := models.User{
			ID:       bson.NewObjectId(),
			Name:     user.Name,
			Username: user.Username,
			Email:    user.Email,
			Password: string(pwdHash),
			Created:  time.Now(),
		}

		if err := newUser.Insert(); err != nil {
			fmt.Printf(" Inserting user failed. If you already have a user, you can disregard this error.\n %v\n", err.Error())
			os.Exit(1)
		}

		fmt.Printf("\n You are all setup %v!\n", user.Name)
	}
	fmt.Printf(" You can login with %v or %v.\n", user.Email, user.Username)

	return 0
}

func readNewline(pre string, stdin *bufio.Reader) string {
	fmt.Print(pre)

	str, _ := stdin.ReadString('\n')
	str = strings.Replace(str, "\n", "", -1)

	return str
}