package main

import (
	"os"
	"fmt"
	log "github.com/Sirupsen/logrus"
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
	log.Info("Checking database connectivity.. Please be patient.")

	if err := db.Connect(); err != nil {
		log.Fatal("\n Cannot connect to database!\n" + err.Error())
	}

	stdin := bufio.NewReader(os.Stdin)

	user := models.User{
		ID: bson.NewObjectId(),
		Username: "admin",
		IsSystemAuditor: false,
		IsSuperUser: true,
		Created: time.Now(),
	}
	// username is optional (default admin)
	username := readNewline("\n > Username (optional, default `admin`): ", stdin)
	if username != "" {
		user.Username = strings.ToLower(username)
	}

	var ouser models.User
	err := db.Users().Find(bson.M{"username": user.Username}).One(&ouser)

	if err == nil {
		// user already exists
		fmt.Printf("\n Welcome back, %v! (a user with this username/email is already set up..)\n\n", ouser.Username)
	} else {
		user.Email = readNewline("\n > Email: ", stdin)
		if user.Email == "" {
			log.Fatal("\n Email is required\n")
			return 1
		}
		user.Email = strings.ToLower(user.Email)

		user.FirstName = readNewline(" > First Name: ", stdin)
		if user.FirstName == "" {
			log.Fatal("\n First Name is required\n")
			return 1
		}

		user.LastName = readNewline(" > Last Name: ", stdin)
		if user.LastName == "" {
			log.Fatal("\n First Lasti is required\n")
			return 1
		}

		user.Password = readNewline(" > Password: ", stdin)
		if user.Password == "" {
			log.Fatal("\n Password is required\n")
			return 1
		}

		pwdHash, _ := bcrypt.GenerateFromPassword([]byte(user.Password), 11)
		user.Password = string(pwdHash)

		if err := db.Users().Insert(user); err != nil {
			fmt.Printf(" Failed to create. If you already setup a user, you can disregard this error.\n %v\n", err.Error())
			os.Exit(1)
		}

		fmt.Printf("\n You are all setup %v!\n", ouser.FirstName + " " + ouser.LastName)
	}
	fmt.Printf(" You can login with `%v`.\n", user.Username)

	return 0
}

func readNewline(pre string, stdin *bufio.Reader) string {
	fmt.Print(pre)

	str, _ := stdin.ReadString('\n')
	str = strings.Replace(str, "\n", "", -1)

	return str
}