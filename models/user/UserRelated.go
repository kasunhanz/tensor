package user

type UserRelated struct {
	AdminOfOrganizations string `json:"admin_of_organizations"`
	Organizations        string `json:"organizations"`
	Roles                string `json:"roles"`
	AccessList           string `json:"access_list"`
	Teams                string `json:"teams"`
	Credentials          string `json:"credentials"`
	ActivityStream       string `json:"activity_stream"`
	Projects             string `json:"projects"`
}

