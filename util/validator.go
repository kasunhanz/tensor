package util

import (
	"reflect"
	"sync"

	"fmt"
	"io"
	"net"
	"regexp"
	"strings"

	"gopkg.in/gin-gonic/gin.v1/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/universal-translator"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/go-playground/validator.v9"
	en_translations "gopkg.in/go-playground/validator.v9/translations/en"
)

const (
	Become           string = "^(sudo|su|pbrun|pfexec|runas|doas|dzdo)$"
	CredentialKind   string = "^(windows|ssh|net|scm|aws|rax|vmware|satellite6|cloudforms|gce|azure|openstack)$"
	ScmType          string = "^(manual|git|hg|svn)$"
	JobType          string = "^(run|check|scan)$"
	ProjectKind      string = "^(ansible|terraform)$"
	TerraformJobType string = "^(plan|apply)$"

	DNSName      string = `^([a-zA-Z0-9]{1}[a-zA-Z0-9_-]{1,62}){1}(\.[a-zA-Z0-9]{1}[a-zA-Z0-9_-]{1,62})*$`
	IP           string = `(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`
	URLSchema    string = `((ftp|tcp|udp|wss?|https?):\/\/)`
	URLUsername  string = `(\S+(:\S*)?@)`
	URLPath      string = `((\/|\?|#)[^\s]*)`
	URLPort      string = `(:(\d{1,5}))`
	URLIP        string = `([1-9]\d?|1\d\d|2[01]\d|22[0-3])(\.(1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.([0-9]\d?|1\d\d|2[0-4]\d|25[0-4]))`
	URLSubdomain string = `((www\.)|([a-zA-Z0-9]([-\.][a-zA-Z0-9]+)*))`
	URL          string = `^` + URLSchema + `?` + URLUsername + `?` + `((` + URLIP + `|(\[` + IP + `\])|(([a-zA-Z0-9]([a-zA-Z0-9-]+)?[a-zA-Z0-9]([-\.][a-zA-Z0-9]+)*)|(` + URLSubdomain + `?))?(([a-zA-Z\x{00a1}-\x{ffff}0-9]+-?-?)*[a-zA-Z\x{00a1}-\x{ffff}0-9]+)(?:\.([a-zA-Z\x{00a1}-\x{ffff}]{1,}))?))` + URLPort + `?` + URLPath + `?$`
)

// use a single instance , it caches struct info
var Uni *ut.UniversalTranslator
var trans ut.Translator

var (
	rxBecome           = regexp.MustCompile(Become)
	rxDNSName          = regexp.MustCompile(DNSName)
	rxURL              = regexp.MustCompile(URL)
	rxCredentialKind   = regexp.MustCompile(CredentialKind)
	rxScmType          = regexp.MustCompile(ScmType)
	rxJobType          = regexp.MustCompile(JobType)
	rxProjectKind      = regexp.MustCompile(ProjectKind)
	rxTerraformJobType = regexp.MustCompile(TerraformJobType)
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
		v.validate = validator.New()
		v.validate.SetTagName("binding")
		// set translator
		en := en.New()
		Uni = ut.New(en, en)
		trans, _ = Uni.GetTranslator("en")
		en_translations.RegisterDefaultTranslations(v.validate, trans)

		// Register custom validator functions
		v.validate.RegisterValidation("become_method", isBecome)
		v.validate.RegisterValidation("dnsname", IsDNSName)
		v.validate.RegisterValidation("iphost", IsHost)
		v.validate.RegisterValidation("credentialkind", IsCredentialKind)
		v.validate.RegisterValidation("naproperty", NaProperty)
		v.validate.RegisterValidation("scmtype", IsScmType)
		v.validate.RegisterValidation("jobtype", IsJobType)
		v.validate.RegisterValidation("project_kind", IsProjectKind)
		v.validate.RegisterValidation("terraform_jobtype", IsTerraformJobType)

		//translations
		// credentialkind
		v.validate.RegisterTranslation("credentialkind", trans, func(ut ut.Translator) error {
			return ut.Add("credentialkind", "{0} must have either one of windows,ssh,net,scm,aws,rax,vmware,satellite6,cloudforms,gce,azure,openstack", true)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("credentialkind", fe.Field())

			return t
		})

		v.validate.RegisterTranslation("project_kind", trans, func(ut ut.Translator) error {
			return ut.Add("project_kind", "{0} must have either one of ansible,terraform", true)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("project_kind", fe.Field())

			return t
		})

		v.validate.RegisterTranslation("become_method", trans, func(ut ut.Translator) error {
			return ut.Add("become_method", "{0} must have either one of sudo,su,pbrun,pfexec,runas,doas,dzdo", true)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("become_method", fe.Field())

			return t
		})
		v.validate.RegisterTranslation("naproperty", trans, func(ut ut.Translator) error {
			return ut.Add("naproperty", "Invalid property, {0}", true)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("naproperty", fe.Field())

			return t
		})
		v.validate.RegisterTranslation("scmtype", trans, func(ut ut.Translator) error {
			return ut.Add("scmtype", "{0} must have either one of manual,git,hg,svn", true)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("scmtype", fe.Field())

			return t
		})
		v.validate.RegisterTranslation("jobtype", trans, func(ut ut.Translator) error {
			return ut.Add("jobtype", "{0} must have either one of run,check,scan", true)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("jobtype", fe.Field())

			return t
		})
		v.validate.RegisterTranslation("terraform_jobtype", trans, func(ut ut.Translator) error {
			return ut.Add("terraform_jobtype", "{0} must have either one of apply,plan", true)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("terraform_jobtype", fe.Field())

			return t
		})
		v.validate.RegisterTranslation("iphost", trans, func(ut ut.Translator) error {
			return ut.Add("iphost", "{0} must have valid hostname or ip address", true)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("iphost", fe.Field())

			return t
		})

		//struct level validations
		v.validate.RegisterStructValidation(CredentialStructLevelValidation, common.Credential{})
		v.validate.RegisterStructValidation(ProjectStructLevelValidation, common.Project{})
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

func isBecome(fl validator.FieldLevel) bool {
	return rxBecome.MatchString(fl.Field().String())
}

// IsDNSName will validate the given string as a DNS name
func IsDNSName(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	if str == "" || len(strings.Replace(str, ".", "", -1)) > 255 {
		// constraints already violated
		return false
	}
	return rxDNSName.MatchString(str)
}

// IsIP checks if a string is either IP version 4 or 6.
func IsIP(fl validator.FieldLevel) bool {
	return net.ParseIP(fl.Field().String()) != nil
}

// IsHost checks if the string is a valid IP (both v4 and v6) or a valid DNS name
func IsHost(fl validator.FieldLevel) bool {
	return IsIP(fl) || IsDNSName(fl)
}

func IsScmType(fl validator.FieldLevel) bool {
	return rxScmType.MatchString(fl.Field().String())
}

func IsJobType(fl validator.FieldLevel) bool {
	return rxJobType.MatchString(fl.Field().String())
}

func IsCredentialKind(fl validator.FieldLevel) bool {
	return rxCredentialKind.MatchString(fl.Field().String())
}

func IsProjectKind(fl validator.FieldLevel) bool {
	return rxProjectKind.MatchString(fl.Field().String())
}
func IsTerraformJobType(fl validator.FieldLevel) bool {
	return rxTerraformJobType.MatchString(fl.Field().String())
}

// fail all
func NaProperty(fl validator.FieldLevel) bool {
	if fl.Field().String() == "" {
		// constraints not violated
		return true
	}

	return false
}

func GetValidationErrors(err error) []string {
	if err == io.EOF {
		return []string{"Request body cannot be empty"}
	}

	if reflect.Ptr == reflect.TypeOf(err).Kind() {
		return []string{"Invalid request body, " + err.Error()}
	}

	// translate all error at once
	if errs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range errs {
			// can translate each error one at a time.
			fmt.Println(e.Translate(trans))

			allerrs := []string{}

			for _, v := range errs.Translate(trans) {
				allerrs = append(allerrs, v)
			}

			return allerrs
		}
	}

	return []string{}
}

// TODO: openstack,azure,gce,
func CredentialStructLevelValidation(sl validator.StructLevel) {

	credentail := sl.Current().Interface().(common.Credential)

	if credentail.Kind == "net" && len(credentail.Username) == 0 {
		sl.ReportError(credentail.Username, "Username", "Username", "required", "")
	}

	if credentail.Kind == "aws" || credentail.Kind == "rax" {
		if len(credentail.Username) == 0 {
			sl.ReportError(credentail.Username, "Username", "Username", "required", "")
		}

		if len(credentail.Password) == 0 {
			sl.ReportError(credentail.Username, "Password", "Password", "required", "")
		}
	}

	if credentail.Kind == "vmware" || credentail.Kind == "satellite6" || credentail.Kind == "cloudforms" {
		if len(credentail.Username) == 0 {
			sl.ReportError(credentail.Username, "Username", "Username", "required", "")
		}

		if len(credentail.Password) == 0 {
			sl.ReportError(credentail.Username, "Password", "Password", "required", "")
		}

		if len(credentail.Host) == 0 {
			sl.ReportError(credentail.Host, "Host", "Password", "required", "")
		}
	}
}

func ProjectStructLevelValidation(sl validator.StructLevel) {
	project := sl.Current().Interface().(common.Project)

	if project.ScmType == "git" || project.ScmType == "svn" || project.ScmType == "hg" {
		if len(project.ScmURL) == 0 {
			sl.ReportError(project.ScmURL, "ScmUrl", "ScmUrl", "required", "")
		}
	}

}
