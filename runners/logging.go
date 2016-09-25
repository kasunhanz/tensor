package runners

import (
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/db"
	"log"
	"time"
)

func (t *AnsibleJob) start() {
	t.job.Status = "running"
	t.job.Started = time.Now()

	c := db.C(db.JOBS)

	d := bson.M{
		"$set": bson.M{
			"status": t.job.Status,
			"failed": false,
			"started": t.job.Started,
		},
	}

	if err := c.UpdateId(t.job.ID, d); err != nil {
		log.Println("Failed to update job status, status was", t.job.Status, err)
	}
}

func (t *AnsibleJob) fail() {
	t.job.Status = "failed"
	t.job.Finished = time.Now()
	t.job.Failed = true

	c := db.C(db.JOBS)

	//get elapsed time in minutes
	diff := t.job.Finished.Sub(t.job.Started)

	d := bson.M{
		"$set": bson.M{
			"status": t.job.Status,
			"failed": t.job.Failed,
			"finished": t.job.Finished,
			"elapsed": diff.Minutes(),
		},
	}

	if err := c.UpdateId(t.job.ID, d); err != nil {
		log.Println("Failed to update job status, status was", t.job.Status, err)
	}

	t.updateProject()
	t.updateJobTemplate()
}

func (t *AnsibleJob) success() {
	t.job.Status = "success"
	t.job.Finished = time.Now()
	t.job.Failed = false

	c := db.C(db.JOBS)

	//get elapsed time in minutes
	diff := t.job.Finished.Sub(t.job.Started)

	d := bson.M{
		"$set": bson.M{
			"status": t.job.Status,
			"failed": t.job.Failed,
			"finished": t.job.Finished,
			"elapsed": diff.Minutes(),
			"stdout_text": t.job.StdoutText,
		},
	}

	if err := c.UpdateId(t.job.ID, d); err != nil {
		log.Println("Failed to update job status, status was", t.job.Status, err)
	}

	t.updateProject()
	t.updateJobTemplate()
}

func (t *AnsibleJob) updateProject() {
	c := db.C(db.PROJECTS)

	d := bson.M{
		"$set": bson.M{
			"last_job_run": t.job.Started,
			"last_job_failed": t.job.Failed,
			"status": t.job.Status,
		},
	}

	if err := c.UpdateId(t.project.ID, d); err != nil {
		log.Println("Failed to update project", err)
	}
}

func (t *AnsibleJob) updateJobTemplate() {
	c := db.C(db.JOB_TEMPLATES)

	d := bson.M{
		"$set": bson.M{
			"last_job_run": t.job.Started,
			"last_job_failed": t.job.Failed,
			"status": t.job.Status,
		},
	}

	if err := c.UpdateId(t.template.ID, d); err != nil {
		log.Println("Failed to update JobTemplate", t.job.Status, err)
	}
}