package util

import (
	"reflect"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"gopkg.in/go-playground/validator.v8"
	"regexp"
)

const (
	domainServerRegexString        = "^[a-zA-Z0-9-]{1,61}\\.[a-zA-Z0-9][a-zA-Z0-9-]{1,61}[a-zA-Z0-9]\\.[a-zA-Z]{2,}$"
	ansibleBecomeMethodRegexString = "^(sudo|su|pbrun|pfexec|runas|doas|dzdo)$"
)

var (
	ansibleBecomeMethodRegex = regexp.MustCompile(ansibleBecomeMethodRegexString)
	domainServerRegex        = regexp.MustCompile(domainServerRegexString)
)

type SpaceValidator struct {
	once     sync.Once
	validate *validator.Validate
}

var _ binding.StructValidator = &SpaceValidator{}

func (v *SpaceValidator) ValidateStruct(obj interface{}) error {
	if kindOfData(obj) == reflect.Struct {
		v.lazyinit()
		if err := v.validate.Struct(obj); err != nil {
			return error(err)
		}
	}
	return nil
}

func (v *SpaceValidator) lazyinit() {
	v.once.Do(func() {
		config := &validator.Config{TagName: "binding"}
		v.validate = validator.New(config)

		// Register custom validator functions
		v.validate.RegisterValidation("ansible_becomemethod", isAnsibleBecomeMethod)
		v.validate.RegisterValidation("domain_server", isDomainServer)
	})
}

func kindOfData(data interface{}) reflect.Kind {
	value := reflect.ValueOf(data)
	valueType := value.Kind()
	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}
	return valueType
}

func isAnsibleBecomeMethod(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return ansibleBecomeMethodRegex.MatchString(field.String())
}

func isDomainServer(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return domainServerRegex.MatchString(field.String())
}
