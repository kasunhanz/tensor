package util

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/securecookie"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
)

var InteractiveSetup bool
var Secrets bool

type MongoDBConfig struct {
	Hosts      []string `yaml:"hosts"`
	Username   string   `yaml:"user"`
	Password   string   `yaml:"pass"`
	DbName     string   `yaml:"name"`
	ReplicaSet string   `yaml:"replica_set"`
}

type RedisConfig struct {
	Host string `yaml:"host"`
}

type configType struct {
	MongoDB             MongoDBConfig `yaml:"mongodb"`

	Redis               RedisConfig `yaml:"redis"`
	// Format `:port_num` eg, :3000
	Port                string `yaml:"port"`

	// Tensor stores projects here
	ProjectsHome        string `yaml:"projects_home"`

	// cookie hashing & encryption
	Salt                string `yaml:"salt"`

	AnsibleJobTimeOut   int `yaml:"ansible_job_timeout"`
	SyncJobTimeOut      int `yaml:"sync_job_timeout"`
	TerraformJobTimeOut int `yaml:"terraform_job_timeout"`

	JWTTimeout          int `ymal:"jwt_timeout"`
	JWTRefreshTimeout   int`ymal:"jwt_refresh_timeout"`
}

var Config *configType

func init() {
	flag.BoolVar(&InteractiveSetup, "setup", false, "perform interactive setup")
	flag.BoolVar(&Secrets, "secrets", false, "generate salt")
	var pwd string
	flag.StringVar(&pwd, "hash", "", "generate hash of given password")

	flag.Parse()

	if len(pwd) > 0 {
		password, _ := bcrypt.GenerateFromPassword([]byte(pwd), 11)
		fmt.Println("Generated password: ", string(password))
		os.Exit(0)
	}
	if Secrets {
		GenerateSalt()
		os.Exit(0)
	}

	if _, err := os.Stat("/etc/tensor.conf"); os.IsNotExist(err) {
		log.Println("Configuration file does not exist")
		Config = &configType{} // initialize empty

	} else {
		conf, err := ioutil.ReadFile("/etc/tensor.conf")

		if err != nil {
			log.Fatal(errors.New("Could not find configuration!\n\n" + err.Error()))
			os.Exit(5)
		}

		if err := yaml.Unmarshal(conf, &Config); err != nil {
			log.Fatal("Invalid Configuration!\n\n" + err.Error())
			os.Exit(6)
		}
	}

	if len(os.Getenv("TENSOR_PORT")) > 0 {
		Config.Port = os.Getenv("TENSOR_PORT")
	} else if len(Config.Port) == 0 {
		Config.Port = ":3000"
	}

	if len(os.Getenv("PROJECTS_HOME")) > 0 {
		Config.ProjectsHome = os.Getenv("PROJECTS_HOME")
	} else if len(Config.ProjectsHome) == 0 {
		Config.ProjectsHome = "/opt/tensor/projects"
	}

	if len(os.Getenv("TENSOR_SALT")) > 0 {
		Config.Salt = os.Getenv("TENSOR_SALT")
	} else if len(Config.Salt) == 0 {
		Config.Salt = "8m86pie1ef8bghbq41ru!de4"
	}

	if len(os.Getenv("TENSOR_ANSIBLE_JOB_TIMEOUT")) > 0 {
		time, _ := strconv.Atoi(os.Getenv("TENSOR_ANSIBLE_JOB_TIMEOUT"))
		Config.AnsibleJobTimeOut = time
	} else if Config.AnsibleJobTimeOut == 0 {
		Config.AnsibleJobTimeOut = 3600
	}

	if len(os.Getenv("TENSOR_TERRAFORM_JOB_TIMEOUT")) > 0 {
		time, _ := strconv.Atoi(os.Getenv("TENSOR_TERRAFORM_JOB_TIMEOUT"))
		Config.TerraformJobTimeOut = time
	} else if Config.TerraformJobTimeOut == 0 {
		Config.TerraformJobTimeOut = 3600
	}

	if len(os.Getenv("TENSOR_SYNC_JOB_TIMEOUT")) > 0 {
		time, _ := strconv.Atoi(os.Getenv("TENSOR_SYNC_JOB_TIMEOUT"))
		Config.SyncJobTimeOut = time
	} else if Config.SyncJobTimeOut == 0 {
		Config.SyncJobTimeOut = 3600
	}

	if len(os.Getenv("TENSOR_JWT_TIMEOUT")) > 0 {
		time, _ := strconv.Atoi(os.Getenv("TENSOR_JWT_TIMEOUT"))
		Config.JWTTimeout = time
	} else if Config.JWTTimeout == 0 {
		Config.JWTTimeout = 3600
	}

	if len(os.Getenv("TENSOR_JWT_REFRESH_TIMEOUT")) > 0 {
		time, _ := strconv.Atoi(os.Getenv("TENSOR_JWT_REFRESH_TIMEOUT"))
		Config.JWTRefreshTimeout = time
	} else if Config.JWTRefreshTimeout == 0 {
		Config.JWTRefreshTimeout = 3600
	}


	if len(os.Getenv("TENSOR_DB_USER")) > 0 {
		Config.MongoDB.Username = os.Getenv("TENSOR_DB_USER")
	}

	if len(os.Getenv("TENSOR_DB_PASSWORD")) > 0 {
		Config.MongoDB.Password = os.Getenv("TENSOR_DB_PASSWORD")
	}

	if len(os.Getenv("TENSOR_DB_NAME")) > 0 {
		Config.MongoDB.DbName = os.Getenv("TENSOR_DB_NAME")
	}

	if len(os.Getenv("TENSOR_DB_REPLICA")) > 0 {
		Config.MongoDB.ReplicaSet = os.Getenv("TENSOR_DB_REPLICA")
	}

	if len(os.Getenv("TENSOR_DB_HOSTS")) > 0 {
		Config.MongoDB.Hosts = strings.Split(os.Getenv("TENSOR_DB_HOSTS"), ";")
	}

	if len(os.Getenv("TENSOR_REDIS_HOST")) > 0 {
		Config.Redis.Host = os.Getenv("TENSOR_REDIS_HOST")
	}

	if _, err := os.Stat(Config.ProjectsHome); os.IsNotExist(err) {
		fmt.Printf(" Running: mkdir -p %v..\n", Config.ProjectsHome)
		if err := os.MkdirAll(Config.ProjectsHome, 0755); err != nil {
			log.Fatal(err)
			os.Exit(7)
		}
	}

}

func GenerateSalt() {
	salt := securecookie.GenerateRandomKey(24)
	fmt.Println("Generated Salt: ", salt)
}
