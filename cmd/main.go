package main

import (
	"os"
	"fmt"
	"log"
	"bufio"
	"strings"
	"gopkg.in/mgo.v2/bson"
	"golang.org/x/crypto/bcrypt"
	"time"
	"bitbucket.pearson.com/apseng/tensor/util"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
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

	var ouser models.User

	userc := db.C(db.USERS)
	err := userc.Find(bson.M{"email": user.Email, "username": user.Username}).One(&ouser)

	if err == nil {
		// user already exists
		fmt.Printf("\n Welcome back, %v! (a user with this username/email is already set up..)\n\n", ouser.FirstName + " " + ouser.LastName)
	} else {
		user.FirstName = readNewline(" > First name: ", stdin)
		user.LastName = readNewline(" > Last name: ", stdin)
		user.Password = readNewline(" > Password: ", stdin)
		user.IsSuperUser = true
		pwdHash, _ := bcrypt.GenerateFromPassword([]byte(user.Password), 11)

		newUser := models.User{
			ID:       bson.NewObjectId(),
			FirstName:     user.FirstName,
			LastName:     user.LastName,
			Username: user.Username,
			Email:    user.Email,
			Password: string(pwdHash),
			Created:  time.Now(),
		}

		if err := userc.Insert(newUser); err != nil {
			fmt.Printf(" Inserting user failed. If you already have a user, you can disregard this error.\n %v\n", err.Error())
			os.Exit(1)
		}

		fmt.Printf("\n You are all setup %v!\n", ouser.FirstName + " " + ouser.LastName)
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