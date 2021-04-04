package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/twinj/uuid"
	"gitlab.com/pbobby001/postit-scheduler/db"
	"gitlab.com/pbobby001/postit-scheduler/pkg/logs"
	"gitlab.com/pbobby001/postit-scheduler/pkg/models"
	"gitlab.com/pbobby001/postit-scheduler/pkg/utils"
	"io/ioutil"
	"net/http"
	"time"
)

func GetSchedule(w http.ResponseWriter, r *http.Request) {
	transactionId := uuid.NewV4().String()
	logs.Logger.Info("Transaction Id:", transactionId)

	tenantNamespace := r.Header.Get("tenant-namespace")
	logs.Logger.Info("Tenant Namespace:", tenantNamespace)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		_ = logs.Logger.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	logs.Logger.Info(string(body))

	var schedule models.PostSchedule
	err = json.Unmarshal(body, &schedule)
	if err != nil {
		_ = logs.Logger.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	logs.Logger.Info("Schedule: ", schedule)

	query := fmt.Sprintf("INSERT INTO general_schedule_table(schedule_id, tenant_namespace, duration_per_post) VALUES ($1, $2, $3)")

	_, err = db.Connection.Exec(
		query,
		&schedule.ScheduleId,
		&tenantNamespace,
		&schedule.Duration,
	)
	if err != nil {
		_ = logs.Logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	facebookPost := make(chan models.SinglePostWithProfiles, 1)
	twitterPost := make(chan models.SinglePostWithProfiles, 1)
	linkedInPost := make(chan models.SinglePostWithProfiles, 1)
	scheduleChan := make(chan *models.PostSchedule)
	postedToFacebook := make(chan bool)
	postedToTwitter := make(chan bool)
	postedToLikedIn := make(chan bool)

	go utils.HibernateSchedule(db.Connection, schedule, tenantNamespace, scheduleChan)
	go utils.SchedulePosts(scheduleChan, postedToFacebook, postedToTwitter, postedToLikedIn, facebookPost, twitterPost, linkedInPost, db.Connection, tenantNamespace)

	if len(schedule.Profiles.Facebook) != 0 {
		go utils.SendPostToFacebook(facebookPost, postedToFacebook, tenantNamespace, schedule.ScheduleId, db.Connection)
	}

	if len(schedule.Profiles.Twitter) != 0 {
		go utils.SendPostToTwitter(twitterPost, postedToTwitter, tenantNamespace, schedule.ScheduleId, db.Connection)
	}

	if len(schedule.Profiles.LinkedIn) != 0 {
		go utils.SendPostToLinkedIn(linkedInPost, postedToLikedIn, tenantNamespace, schedule.ScheduleId, db.Connection)
	}

	var response = models.StandardResponse{
		Data: models.Data{
			Id:        transactionId,
			UiMessage: "Schedule received and being worked on",
		},
		Meta: models.Meta{
			Timestamp:     time.Now(),
			TransactionId: transactionId,
			TraceId:       "",
			Status:        "SUCCESS",
		},
	}

	_ = json.NewEncoder(w).Encode(&response)
}
