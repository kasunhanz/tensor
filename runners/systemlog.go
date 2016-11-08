package runners

import (
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/db"
	log "github.com/Sirupsen/logrus"
	"time"
	"bitbucket.pearson.com/apseng/tensor/models"
)

const _CTX_UPDATE_JOB = "update_job"

func (t *SystemJob) start() {
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

func (t *SystemJob) fail() {
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
}

func (t *SystemJob) jobCancel() {
	t.Job.Status = "canceled"
	t.Job.Finished = time.Now()
	t.Job.Failed = false

	//get elapsed time in minutes
	diff := t.Job.Finished.Sub(t.Job.Started)

	d := bson.M{
		"$set": bson.M{
			"status": t.Job.Status,
			"failed": t.Job.Failed,
			"cancel_flag": true,
			"finished": t.Job.Finished,
			"elapsed": diff.Minutes(),
			"result_stdout": "stdout capture is missing",
			"job_explanation": "Job Cancelled",
			"job_args": t.Job.JobARGS,
			"job_env": t.Job.JobENV,
			"job_cwd": t.Job.JobCWD,
		},
	}

	if err := db.Jobs().UpdateId(t.Job.ID, d); err != nil {
		log.Println("Failed to update job status, status was", t.Job.Status, err)
	}

	t.updateProject()
}

func (t *SystemJob) success() {
	t.Job.Status = "successful"
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
}

func (t *SystemJob) updateProject() {

	d := bson.M{
		"$set": bson.M{
			"last_updated": t.Job.Finished,
			"last_update_failed": t.Job.Failed,
			"status": t.Job.Status,
		},
	}

	if err := db.Projects().UpdateId(t.Job.ProjectID, d); err != nil {
		log.Println("Failed to update project", err)
	}
}

func addSystemActivity(crdID bson.ObjectId, userID bson.ObjectId, desc string) {

	a := models.Activity{
		ID: bson.NewObjectId(),
		ActorID: userID,
		Type: _CTX_UPDATE_JOB,
		ObjectID: crdID,
		Description: desc,
		Created: time.Now(),
	}

	if err := db.ActivityStream().Insert(a); err != nil {
		log.Println("Failed to add new Activity", err)
	}
}