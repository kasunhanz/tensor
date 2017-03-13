package validate

import (
	"reflect"
	"sync"

	"fmt"
	"io"
	"net"
	"regexp"
	"strings"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/universal-translator"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/gin-gonic/gin.v1/binding"
	"gopkg.in/go-playground/validator.v9"
	en_translations "gopkg.in/go-playground/validator.v9/translations/en"
)

const (
	Become           string = "^(sudo|su|pbrun|pfexec|runas|doas|dzdo)$"
	CredentialKind   string = "^(windows|ssh|net|scm|aws|rax|vmware|satellite6|cloudforms|gce|azure|openstack)$"
	ScmType          string = "^(manual|git|hg|svn)$"
	JobType          string = "^(run|check|scan)$"
	ProjectKind      string = "^(ansible|terraform)$"
	TerraformJobType string = "^(plan|apply|destroy|destroy_plan)$"
	ResourceType     string = "^(credential|organization|team|project|job_template|terraform_job_template|inventory)$"

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
	rxResourceType     = regexp.MustCompile(ResourceType)
)

type Validator struct {
	once     sync.Once
	validate *validator.Validate
}

var _ binding.StructValidator = &Validator{}

func (v *Validator) ValidateStruct(obj interface{}) error {
	if kindOfData(obj) == reflect.Struct {
		v.lazyinit()
		if err := v.validate.Struct(obj); err != nil {
			return error(err)
		}
	}
	return nil
}

func (v *Validator) lazyinit() {
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
		v.validate.RegisterValidation("dnsname", isDNSName)
		v.validate.RegisterValidation("iphost", isHost)
		v.validate.RegisterValidation("credential_kind", isCredentialKind)
		v.validate.RegisterValidation("naproperty", naProperty)
		v.validate.RegisterValidation("scmtype", isScmType)
		v.validate.RegisterValidation("jobtype", isJobType)
		v.validate.RegisterValidation("project_kind", isProjectKind)
		v.validate.RegisterValidation("terraform_jobtype", isTerraformJobType)
		v.validate.RegisterValidation("resource_type", isResourceType)

		//translations
		v.validate.RegisterTranslation("credential_kind", trans, func(ut ut.Translator) error {
			return ut.Add("credential_kind", "{0} must have either one of windows,ssh,net,scm,aws,rax,vmware,satellite6,cloudforms,gce,azure,openstack", true)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("credential_kind", fe.Field())

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

		v.validate.RegisterTranslation("resource_type", trans, func(ut ut.Translator) error {
			return ut.Add("resource_type", "{0} must have either one of credential,organization,team,project,job_template,terraform_job_template,inventory", true)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("resource_type", fe.Field())

			return t
		})

		//struct level validations
		v.validate.RegisterStructValidation(credentialStructLevelValidation, common.Credential{})
		v.validate.RegisterStructValidation(projectStructLevelValidation, common.Project{})
		v.validate.RegisterStructValidation(roleObjStructLevelValidation, common.RoleObj{})
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
func isDNSName(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	if str == "" || len(strings.Replace(str, ".", "", -1)) > 255 {
		// constraints already violated
		return false
	}
	return rxDNSName.MatchString(str)
}

// IsIP checks if a string is either IP version 4 or 6.
func isIP(fl validator.FieldLevel) bool {
	return net.ParseIP(fl.Field().String()) != nil
}

// IsHost checks if the string is a valid IP (both v4 and v6) or a valid DNS name
func isHost(fl validator.FieldLevel) bool {
	return isIP(fl) || isDNSName(fl)
}

func isScmType(fl validator.FieldLevel) bool {
	return rxScmType.MatchString(fl.Field().String())
}

func isJobType(fl validator.FieldLevel) bool {
	return rxJobType.MatchString(fl.Field().String())
}

func isCredentialKind(fl validator.FieldLevel) bool {
	return rxCredentialKind.MatchString(fl.Field().String())
}

func isProjectKind(fl validator.FieldLevel) bool {
	return rxProjectKind.MatchString(fl.Field().String())
}
func isTerraformJobType(fl validator.FieldLevel) bool {
	return rxTerraformJobType.MatchString(fl.Field().String())
}

func isResourceType(fl validator.FieldLevel) bool {
	return rxResourceType.MatchString(fl.Field().String())
}

// fail all
func naProperty(fl validator.FieldLevel) bool {
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

func credentialStructLevelValidation(sl validator.StructLevel) {

	credential := sl.Current().Interface().(common.Credential)

	if credential.Kind == common.CredentialKindNET && len(credential.Username) == 0 {
		sl.ReportError(credential.Username, "Username", "Username", "required", "")
	}

	if credential.Kind == common.CredentialKindAWS {
		if len(credential.Secret) == 0 {
			sl.ReportError(credential.Secret, "Secret", "Secret Access Key", "required", "")
		}

		if len(credential.Client) == 0 {
			sl.ReportError(credential.Client, "Client", "Secret Access ID", "required", "")
		}
	}

	if credential.Kind == common.CredentialKindRAX {
		if len(credential.Username) == 0 {
			sl.ReportError(credential.Username, "Username", "Username", "required", "")
		}

		if len(credential.Secret) == 0 {
			sl.ReportError(credential.Secret, "Secret", "API Key", "required", "")
		}
	}

	if credential.Kind == common.CredentialKindGCE {
		if len(credential.Email) == 0 {
			sl.ReportError(credential.Email, "Email", "Email", "required", "")
		}

		if len(credential.Project) == 0 {
			sl.ReportError(credential.Project, "Project", "Project", "required", "")
		}

		if len(credential.SSHKeyData) == 0 {
			sl.ReportError(credential.SSHKeyData, "SSH Key Data", "SSH Key Data", "required", "")
		}
	}

	if credential.Kind == common.CredentialKindAZURE {
		if len(credential.Username) > 0 {
			if len(credential.Username) == 0 {
				sl.ReportError(credential.Username, "Username", "Azure AD User", "required", "")
			}

			if len(credential.Password) == 0 {
				sl.ReportError(credential.Password, "Password", "Azure AD Password", "required", "")
			}
		} else {
			if len(credential.Secret) == 0 {
				sl.ReportError(credential.Secret, "Secret", "Azure Secret", "required", "")
			}

			if len(credential.Client) == 0 {
				sl.ReportError(credential.Client, "Client", "Azure Client ID", "required", "")
			}
			if len(credential.Tenant) == 0 {
				sl.ReportError(credential.Tenant, "Tenant", "Azure Tenant", "required", "")
			}
		}

		if len(credential.Subscription) == 0 {
			sl.ReportError(credential.Subscription, "Subscription", "Azure Subscription", "required", "")
		}
	}
}

func projectStructLevelValidation(sl validator.StructLevel) {
	project := sl.Current().Interface().(common.Project)

	if project.ScmType == "git" || project.ScmType == "svn" || project.ScmType == "hg" {
		if len(project.ScmURL) == 0 {
			sl.ReportError(project.ScmURL, "Scm Url", "ScmUrl", "required", "")
		}
	}

}

func roleObjStructLevelValidation(sl validator.StructLevel) {
	roleobj := sl.Current().Interface().(common.RoleObj)

	switch roleobj.ResourceType {
	case "credential":
		{
			if roleobj.Role != "admin" && roleobj.Role != "use" {
				sl.ReportError(roleobj.Role, "Role", "Role", "Role must be either one of admin,use", "")
			}
		}
	case "organization":
		{
			if roleobj.Role != "admin" && roleobj.Role != "auditor" && roleobj.Role != "member" {
				sl.ReportError(roleobj.Role, "Role", "Role", "Role must be either one of admin,auditor,member", "")
			}
		}
	case "team":
		{
			if roleobj.Role != "admin" && roleobj.Role != "member" {
				sl.ReportError(roleobj.Role, "Role", "Role", "Role must be either one of admin,member", "")
			}
		}
	case "project", "inventory":
		{
			if roleobj.Role != "admin" && roleobj.Role != "update" && roleobj.Role != "use" {
				sl.ReportError(roleobj.Role, "Role", "Role", "Role must be either one of admin,update,use", "")
			}
		}
	case "job_template", "terraform_job_template":
		{
			if roleobj.Role != "admin" && roleobj.Role != "execute" {
				sl.ReportError(roleobj.Role, "Role", "Role", "Role must be either one of admin,execute", "")
			}
		}
	}
}
