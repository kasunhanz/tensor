package terraform

import (
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/exec/types"
	"github.com/pearsonappeng/tensor/models/common"
)

func start(t *types.TerraformJob) {
	t.Job.Status = "running"
	t.Job.Started = time.Now()

	d := bson.M{
		"$set": bson.M{
			"status":  t.Job.Status,
			"failed":  false,
			"started": t.Job.Started,
		},
	}

	if err := db.TerrafromJobs().UpdateId(t.Job.ID, d); err != nil {
		logrus.WithFields(logrus.Fields{
			"Status": t.Job.Status,
			"Error":  err,
		}).Errorln("Failed to update job status")
	}
}

func status(t *types.TerraformJob, s string) {
	t.Job.Status = s
	d := bson.M{
		"$set": bson.M{
			"status": t.Job.Status,
		},
	}

	if err := db.TerrafromJobs().UpdateId(t.Job.ID, d); err != nil {
		logrus.WithFields(logrus.Fields{
			"Status": t.Job.Status,
			"Error":  err,
		}).Errorln("Failed to update job status")
	}
}

func jobFail(t *types.TerraformJob) {
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

	if err := db.TerrafromJobs().UpdateId(t.Job.ID, d); err != nil {
		logrus.WithFields(logrus.Fields{
			"Status": t.Job.Status,
			"Error":  err,
		}).Errorln("Failed to update job status")
	}

	updateProject(t)
	updateJobTemplate(t)
}

func jobCancel(t *types.TerraformJob) {
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

	if err := db.TerrafromJobs().UpdateId(t.Job.ID, d); err != nil {
		logrus.WithFields(logrus.Fields{
			"Status": t.Job.Status,
			"Error":  err,
		}).Errorln("Failed to update job status")
	}

	updateProject(t)
	updateJobTemplate(t)
}

func jobError(t *types.TerraformJob) {
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

	if err := db.TerrafromJobs().UpdateId(t.Job.ID, d); err != nil {
		logrus.WithFields(logrus.Fields{
			"Status": t.Job.Status,
			"Error":  err,
		}).Errorln("Failed to update job status")
	}

	updateProject(t)
	updateJobTemplate(t)
}

func jobSuccess(t *types.TerraformJob) {
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

	if err := db.TerrafromJobs().UpdateId(t.Job.ID, d); err != nil {
		logrus.WithFields(logrus.Fields{
			"Status": t.Job.Status,
			"Error":  err,
		}).Errorln("Failed to update job status")
	}

	updateProject(t)
	updateJobTemplate(t)
}

func updateProject(t *types.TerraformJob) {
	d := bson.M{
		"$set": bson.M{
			"last_job_run":    t.Job.Started,
			"last_job_failed": t.Job.Failed,
			"status":          t.Job.Status,
		},
	}
	if err := db.Projects().UpdateId(t.Project.ID, d); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
		}).Errorln("Failed to update project")
	}
}

func updateJobTemplate(t *types.TerraformJob) {
	d := bson.M{
		"$set": bson.M{
			"last_job_run":    t.Job.Started,
			"last_job_failed": t.Job.Failed,
			"status":          t.Job.Status,
		},
	}

	if err := db.TerrafromJobTemplates().UpdateId(t.Template.ID, d); err != nil {
		logrus.WithFields(logrus.Fields{
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
		logrus.WithFields(logrus.Fields{
			"Error": err,
		}).Errorln("Failed to add new Activity")
	}
}
