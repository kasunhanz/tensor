package models

import "time"

// AddHocTask is an exported type that
// is used as database model for record
// add hoc command execution details
type AddHocTask struct {
	ID          int `db:"id" json:"id"`
	AccessKeyID int `db:"access_key_id" json:"access_key_id"`

	Status      string `db:"status" json:"status"`
	Debug       bool   `db:"debug" json:"debug"`

	Module      string `db:"module" json:"module"`
	Arguments   string `db:"arguments" json:"arguments"`
	ExtraVars   string `db:"extra_vars" json:"extra_vars"`
	Forks       int    `db:"forks" json:"forks"`
	Inventory   string `db:"inventory" json:"inventory"`
	Connection  string `db:"connection" json:"connection"`
	Timeout     int    `db:"timeout" json:"timeout"`

	Created     time.Time  `db:"created" json:"created"`
	Start       *time.Time `db:"start" json:"start"`
	End         *time.Time `db:"end" json:"end"`
}

// AddHocTaskOutput is an exported type that
// is used as database model for record
// add command database output
type AddHocTaskOutput struct {
	TaskID int       `db:"task_id" json:"task_id"`
	Task   string    `db:"task" json:"task"`
	Time   time.Time `db:"time" json:"time"`
	Output string    `db:"output" json:"output"`
}
