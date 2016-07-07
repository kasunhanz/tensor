package util

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"github.com/gorilla/securecookie"
	"golang.org/x/crypto/bcrypt"
	"errors"
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

var Cookie *securecookie.SecureCookie
var InteractiveSetup bool
var Secrets bool

type MongoDBConfig struct {
	Hosts      []string `yaml:"hosts"`
	Username   string `yaml:"user"`
	Password   string `yaml:"pass"`
	DbName     string `yaml:"name"`
	ReplicaSet string `yaml:"replica_set"`
}

type configType struct {
	MongoDB          MongoDBConfig `yaml:"mongodb"`
	// Format `:port_num` eg, :3000
	Port             string `yaml:"port"`

	// HilbertSpace stores projects here
	TmpPath          string `yaml:"tmp_path"`

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

	conf, err := ioutil.ReadFile("/etc/hilbert_space.yaml")

	if err != nil {
		panic(errors.New("Cannot Find configuration!\n\n" + err.Error()))
	}

	if err := yaml.Unmarshal(conf, &Config); err != nil {
		panic("Invalid Configuration!\n\n" + err.Error())
	}

	if len(os.Getenv("PORT")) > 0 {
		Config.Port = ":" + os.Getenv("PORT")
	}
	if len(Config.Port) == 0 {
		Config.Port = ":3000"
	}

	if len(Config.TmpPath) == 0 {
		Config.TmpPath = "/tmp/hilbertspace"
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
			panic(err)
		}
	}

}

func GenerateCookieSecrets() {
	hash := securecookie.GenerateRandomKey(32)
	encryption := securecookie.GenerateRandomKey(32)

	fmt.Println("Generated Hash: ", base64.StdEncoding.EncodeToString(hash))
	fmt.Println("Generated Encryption: ", base64.StdEncoding.EncodeToString(encryption))
}