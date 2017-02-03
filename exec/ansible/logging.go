package ansible

import (
	"time"

	"gopkg.in/mgo.v2/bson"

	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/exec/types"
)

func start(t types.AnsibleJob) {
	t.Job.Status = "running"
	t.Job.Started = time.Now()

	d := bson.M{
		"$set": bson.M{
			"status":  t.Job.Status,
			"failed":  false,
			"started": t.Job.Started,
		},
	}

	if err := db.Jobs().UpdateId(t.Job.ID, d); err != nil {
		log.WithFields(log.Fields{
			"Status": t.Job.Status,
			"Error":  err,
		}).Errorln("Failed to update job status")
	}
}

func status(t types.AnsibleJob, s string) {
	t.Job.Status = s
	d := bson.M{
		"$set": bson.M{
			"status": t.Job.Status,
		},
	}

	if err := db.Jobs().UpdateId(t.Job.ID, d); err != nil {
		log.WithFields(log.Fields{
			"Status": t.Job.Status,
			"Error":  err,
		}).Errorln("Failed to update job status")
	}
}

func jobFail(t types.AnsibleJob) {
	t.Job.Status = "failed"
	t.Job.Finished = time.Now()
	t.Job.Failed = true

	//get elapsed time in minutes
	diff := t.Job.Finished.Sub(t.Job.Started)

	d := bson.M{
		"$set": bson.M{
			"status":          t.Job.Status,
			"failed":          t.Job.Failed,
			"finished":        t.Job.Finished,
			"elapsed":         diff.Minutes(),
			"result_stdout":   t.Job.ResultStdout,
			"job_explanation": t.Job.JobExplanation,
			"job_args":        t.Job.JobARGS,
			"job_env":         t.Job.JobENV,
			"job_cwd":         t.Job.JobCWD,
		},
	}

	if err := db.Jobs().UpdateId(t.Job.ID, d); err != nil {
		log.WithFields(log.Fields{
			"Status": t.Job.Status,
			"Error":  err,
		}).Errorln("Failed to update job status")
	}

	updateProject(t)
	updateJobTemplate(t)
}

func jobCancel(t types.AnsibleJob) {
	t.Job.Status = "canceled"
	t.Job.Finished = time.Now()
	t.Job.Failed = false

	//get elapsed time in minutes
	diff := t.Job.Finished.Sub(t.Job.Started)

	d := bson.M{
		"$set": bson.M{
			"status":          t.Job.Status,
			"cancel_flag":     true,
			"failed":          t.Job.Failed,
			"finished":        t.Job.Finished,
			"elapsed":         diff.Minutes(),
			"result_stdout":   "stdout capture is missing",
			"job_explanation": "Job Cancelled",
			"job_args":        t.Job.JobARGS,
			"job_env":         t.Job.JobENV,
			"job_cwd":         t.Job.JobCWD,
		},
	}

	if err := db.Jobs().UpdateId(t.Job.ID, d); err != nil {
		log.WithFields(log.Fields{
			"Status": t.Job.Status,
			"Error":  err,
		}).Errorln("Failed to update job status")
	}

	updateProject(t)
	updateJobTemplate(t)
}

func jobError(t types.AnsibleJob) {
	t.Job.Status = "error"
	t.Job.Finished = time.Now()
	t.Job.Failed = true

	//get elapsed time in minutes
	diff := t.Job.Finished.Sub(t.Job.Started)

	d := bson.M{
		"$set": bson.M{
			"status":          t.Job.Status,
			"failed":          t.Job.Failed,
			"finished":        t.Job.Finished,
			"elapsed":         diff.Minutes(),
			"result_stdout":   t.Job.ResultStdout,
			"job_explanation": t.Job.JobExplanation,
			"job_args":        t.Job.JobARGS,
			"job_env":         t.Job.JobENV,
			"job_cwd":         t.Job.JobCWD,
		},
	}

	if err := db.Jobs().UpdateId(t.Job.ID, d); err != nil {
		log.WithFields(log.Fields{
			"Status": t.Job.Status,
			"Error":  err,
		}).Errorln("Failed to update job status")
	}

	updateProject(t)
	updateJobTemplate(t)
}

func jobSuccess(t types.AnsibleJob) {
	t.Job.Status = "successful"
	t.Job.Finished = time.Now()
	t.Job.Failed = false

	//get elapsed time in minutes
	diff := t.Job.Finished.Sub(t.Job.Started)

	d := bson.M{
		"$set": bson.M{
			"status":          t.Job.Status,
			"failed":          t.Job.Failed,
			"finished":        t.Job.Finished,
			"elapsed":         diff.Minutes(),
			"result_stdout":   t.Job.ResultStdout,
			"job_explanation": t.Job.JobExplanation,
			"job_args":        t.Job.JobARGS,
			"job_env":         t.Job.JobENV,
			"job_cwd":         t.Job.JobCWD,
		},
	}

	if err := db.Jobs().UpdateId(t.Job.ID, d); err != nil {
		log.WithFields(log.Fields{
			"Status": t.Job.Status,
			"Error":  err,
		}).Errorln("Failed to update job status")
	}

	updateProject(t)
	updateJobTemplate(t)
}

func updateProject(t types.AnsibleJob) {
	d := bson.M{
		"$set": bson.M{
			"last_job_run":    t.Job.Started,
			"last_job_failed": t.Job.Failed,
			"status":          t.Job.Status,
		},
	}
	if err := db.Projects().UpdateId(t.Project.ID, d); err != nil {
		log.WithFields(log.Fields{
			"Error": err,
		}).Errorln("Failed to update project")
	}
}

func updateJobTemplate(t types.AnsibleJob) {
	d := bson.M{
		"$set": bson.M{
			"last_job_run":    t.Job.Started,
			"last_job_failed": t.Job.Failed,
			"status":          t.Job.Status,
		},
	}

	if err := db.JobTemplates().UpdateId(t.Template.ID, d); err != nil {
		log.WithFields(log.Fields{
			"Status": t.Job.Status,
			"Error":  err,
		}).Errorln("Failed to update JobTemplate")
	}
}

func addActivity(crdID bson.ObjectId, userID bson.ObjectId, desc string, jobtype string) {

	a := common.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     userID,
		Type:        jobtype,
		ObjectID:    crdID,
		Description: desc,
		Created:     time.Now(),
	}

	if err := db.ActivityStream().Insert(a); err != nil {
		log.WithFields(log.Fields{
			"Error": err,
		}).Errorln("Failed to add new Activity")
	}
}
