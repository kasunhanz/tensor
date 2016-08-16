package util

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/securecookie"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"log"
)

var Cookie *securecookie.SecureCookie
var InteractiveSetup bool
var Secrets bool

type MongoDBConfig struct {
	Hosts      []string `yaml:"hosts"`
	Username   string   `yaml:"user"`
	Password   string   `yaml:"pass"`
	DbName     string   `yaml:"name"`
	ReplicaSet string   `yaml:"replica_set"`
}

type configType struct {
	MongoDB MongoDBConfig `yaml:"mongodb"`
	// Format `:port_num` eg, :3000
	Port string `yaml:"port"`
	UiPort string `yaml:"uiport"`

	// Tensor stores projects here
	TmpPath string `yaml:"tmp_path"`

	// cookie hashing & encryption
	CookieHash       string `yaml:"cookie_hash"`
	CookieEncryption string `yaml:"cookie_encryption"`
}

var Config *configType

func init() {
	flag.BoolVar(&InteractiveSetup, "setup", false, "perform interactive setup")

	flag.BoolVar(&Secrets, "secrets", false, "generate cookie secrets")

	var pwd string
	flag.StringVar(&pwd, "hash", "", "generate hash of given password")

	flag.Parse()

	if len(pwd) > 0 {
		password, _ := bcrypt.GenerateFromPassword([]byte(pwd), 11)
		fmt.Println("Generated password: ", string(password))

		os.Exit(0)
	}
	if Secrets {
		GenerateCookieSecrets()

		os.Exit(0)
	}

	conf, err := ioutil.ReadFile("/etc/tensor.conf")

	if err != nil {
		log.Fatal(errors.New("Could not find configuration!\n\n" + err.Error()))
		os.Exit(5)
	}

	if err := yaml.Unmarshal(conf, &Config); err != nil {
		log.Fatal("Invalid Configuration!\n\n" + err.Error())
		os.Exit(6)
	}

	if len(os.Getenv("PORT")) > 0 {
		Config.Port = ":" + os.Getenv("PORT")
	}
	if len(Config.Port) == 0 {
		Config.Port = ":3000"
	}

	if len(Config.UiPort) == 0 {
		Config.Port = ":8080"
	}

	if len(Config.TmpPath) == 0 {
		Config.TmpPath = "/tmp/tensor"
	}

	var encryption []byte
	encryption = nil

	hash, _ := base64.StdEncoding.DecodeString(Config.CookieHash)
	if len(Config.CookieEncryption) > 0 {
		encryption, _ = base64.StdEncoding.DecodeString(Config.CookieEncryption)
	}

	Cookie = securecookie.New(hash, encryption)

	if _, err := os.Stat(Config.TmpPath); os.IsNotExist(err) {
		fmt.Printf(" Running: mkdir -p %v..\n", Config.TmpPath)
		if err := os.MkdirAll(Config.TmpPath, 0755); err != nil {
			log.Fatal(err)
			os.Exit(7)
		}
	}

}

func GenerateCookieSecrets() {
	hash := securecookie.GenerateRandomKey(32)
	encryption := securecookie.GenerateRandomKey(32)

	fmt.Println("Generated Hash: ", base64.StdEncoding.EncodeToString(hash))
	fmt.Println("Generated Encryption: ", base64.StdEncoding.EncodeToString(encryption))
}
