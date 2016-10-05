package roles

//Important: if you are adding roles to team which means you are adding user to that team
const (
	// organization
	ORGANIZATION_ADMIN = "admin"
	ORGANIZATION_AUDITOR = "auditor"
	ORGANIZATION_MEMBER = "member"
	ORGANIZATION_READ = "read"

	// credential
	CREDENTIAL_ADMIN = "admin"
	CREDENTIAL_READ = "read"
	CREDENTIAL_USE = "use"

	// project
	PROJECT_ADMIN = "admin"
	PROJECT_USE = "use"
	PROJECT_UPDATE = "update"

	// inventory
	INVENTORY_ADMIN = "admin"
	INVENTORY_USE = "use"
	INVENTORY_ADD_HOC = "add_hoc"
	INVENTORY_UPDATE = "update"

	//job template
	JOB_TEMPLATE_ADMIN = "admin"
	JOB_TEMPLATE_EXECUTE = "execute"

	//job
	JOB_ADMIN = "admin"
	JOB_EXECUTE = "execute"

	//Teams
	TEAM_ADMIN = "admin"
	TEAM_MEMBER = "member"
	TEAM_READ = "read"
)