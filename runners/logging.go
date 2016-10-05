package runners

import (
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/db"
	"log"
	"time"
	"bitbucket.pearson.com/apseng/tensor/models"
)

const _CTX_JOB = "job"

func (t *AnsibleJob) start() {
	t.Job.Status = "running"
	t.Job.Started = time.Now()

	d := bson.M{
		"$set": bson.M{
			"status": t.Job.Status,
			"failed": false,
			"started": t.Job.Started,
		},
	}

	if err := db.Jobs().UpdateId(t.Job.ID, d); err != nil {
		log.Println("Failed to update job status, status was", t.Job.Status, err)
	}
}

func (t *AnsibleJob) fail() {
	t.Job.Status = "failed"
	t.Job.Finished = time.Now()
	t.Job.Failed = true

	//get elapsed time in minutes
	diff := t.Job.Finished.Sub(t.Job.Started)

	d := bson.M{
		"$set": bson.M{
			"status": t.Job.Status,
			"failed": t.Job.Failed,
			"finished": t.Job.Finished,
			"elapsed": diff.Minutes(),
			"result_stdout": t.Job.ResultStdout,
			"job_explanation": t.Job.JobExplanation,
			"job_args": t.Job.JobARGS,
			"job_env": t.Job.JobENV,
			"job_cwd": t.Job.JobCWD,
		},
	}

	if err := db.Jobs().UpdateId(t.Job.ID, d); err != nil {
		log.Println("Failed to update job status, status was", t.Job.Status, err)
	}

	t.updateProject()
	t.updateJobTemplate()
}

func (t *AnsibleJob) success() {
	t.Job.Status = "success"
	t.Job.Finished = time.Now()
	t.Job.Failed = false


	//get elapsed time in minutes
	diff := t.Job.Finished.Sub(t.Job.Started)

	d := bson.M{
		"$set": bson.M{
			"status": t.Job.Status,
			"failed": t.Job.Failed,
			"finished": t.Job.Finished,
			"elapsed": diff.Minutes(),
			"result_stdout": t.Job.ResultStdout,
			"job_explanation": t.Job.JobExplanation,
			"job_args": t.Job.JobARGS,
			"job_env": t.Job.JobENV,
			"job_cwd": t.Job.JobCWD,
		},
	}

	if err := db.Jobs().UpdateId(t.Job.ID, d); err != nil {
		log.Println("Failed to update job status, status was", t.Job.Status, err)
	}

	t.updateProject()
	t.updateJobTemplate()
}

func (t *AnsibleJob) updateProject() {

	d := bson.M{
		"$set": bson.M{
			"last_job_run": t.Job.Started,
			"last_job_failed": t.Job.Failed,
			"status": t.Job.Status,
		},
	}

	if err := db.Projects().UpdateId(t.Project.ID, d); err != nil {
		log.Println("Failed to update project", err)
	}
}

func (t *AnsibleJob) updateJobTemplate() {

	d := bson.M{
		"$set": bson.M{
			"last_job_run": t.Job.Started,
			"last_job_failed": t.Job.Failed,
			"status": t.Job.Status,
		},
	}

	if err := db.JobTemplates().UpdateId(t.Template.ID, d); err != nil {
		log.Println("Failed to update JobTemplate", t.Job.Status, err)
	}
}

func addActivity(crdID bson.ObjectId, userID bson.ObjectId, desc string) {

	a := models.Activity{
		ID: bson.NewObjectId(),
		ActorID: userID,
		Type: _CTX_JOB,
		ObjectID: crdID,
		Description: desc,
		Created: time.Now(),
	}

	if err := db.ActivityStream().Insert(a); err != nil {
		log.Println("Failed to add new Activity", err)
	}
}