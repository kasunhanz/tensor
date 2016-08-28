package organization

type OrgRelated struct {
	CreatedBy               string `json:"created_by"`
	ModifiedBy              string `json:"modified_by"`
	NotificationTempError   string `json:"notification_templates_error"`
	NotificationTempSuccess string `json:"notification_templates_success"`
	Users                   string `json:"users"`
	ObjectRoles             string `json:"object_roles"`
	NotificationTempAny     string `json:"notification_templates_any"`
	Teams                   string `json:"teams"`
	AccessList              string `json:"access_list"`
	NotificationTemplates   string `json:"notification_templates"`
	Admins                  string `json:"admins"`
	Credentials             string `json:"credentials"`
	Inventories             string `json:"inventories"`
	ActivityStream          string `json:"activity_stream"`
	Projects                string `json:"projects"`
}

