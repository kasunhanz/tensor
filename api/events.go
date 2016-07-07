package api

import (
	"github.com/gin-gonic/gin"
)

func getEvents(c *gin.Context) {
	/*user := c.MustGet("user").(*models.User)

	q := squirrel.Select("event.*, p.name as project_name").
	From("event").
	LeftJoin("project as p on event.project_id=p.id").
	OrderBy("created desc")

	col := database.MongoDb.C("event")

	aggrigrate := []bson.M{
		{"$lookup" : bson.M{
			"from":"project",
			"localField":"project_id",
			"foreignField":"_id",
			"as": "project",
		}},
		{"$match": bson.M{
			"$project_template._id":project.ID,
		}},
		{"$sort": bson.M{
			"created":-1,
		}},
	}

	projectObj, exists := c.Get("project")
	if exists == true {
		// limit query to project
		project := projectObj.(models.Project)

		aggrigrate = append([]bson.M{{"$match": bson.M{
			"project_id":project.ID,
		}}}, aggrigrate...)

		q = q.Where("event.project_id=?", project.ID)
	} else {

		aggrigrate = append([]bson.M{
			{"$lookup" : bson.M{
				"from":"project__user",
				"localField":"project_id",
				"foreignField":"_id",
				"as": "project",
			}},
		}, aggrigrate...)

		q = q.LeftJoin("project__user as pu on pu.project_id=p.id").
		Where("p.id IS NULL or pu.user_id=?", user.ID)
	}

	var events []models.Event

	query, args, _ := q.ToSql()
	if _, err := database.Mysql.Select(&events, query, args...); err != nil {
		panic(err)
	}

	for i, evt := range events {
		if evt.ObjectID == nil || evt.ObjectType == nil {
			continue
		}

		var q squirrel.SelectBuilder

		switch *evt.ObjectType {
		case "task":
			q = squirrel.Select("tpl.playbook as name").
			From("task").
			Join("project__template as tpl on task.template_id=tpl.id")
		default:
			continue
		}

		query, args, _ := q.ToSql()
		name, err := database.Mysql.SelectNullStr(query, args...)
		if err != nil {
			panic(err)
		}

		if name.Valid == true {
			events[i].ObjectName = name.String
		}
	}

	c.JSON(200, events)*/
	c.JSON(200, nil)
}
