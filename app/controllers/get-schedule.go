package controllers

import (
	"encoding/json"
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
		go utils.SendPostToFacebook(facebookPost, postedToFacebook, tenantNamespace, db.Connection)
	}

	if len(schedule.Profiles.Twitter) != 0 {
		go utils.SendPostToTwitter(twitterPost, postedToTwitter, tenantNamespace, db.Connection)
	}

	if len(schedule.Profiles.LinkedIn) != 0 {
		go utils.SendPostToLinkedIn(linkedInPost, postedToLikedIn, tenantNamespace, db.Connection)
	}

	var response = models.StandardResponse {
		Data: models.Data {
			Id:        transactionId,
			UiMessage: "Schedule received and being worked on",
		},
		Meta: models.Meta {
			Timestamp:     time.Now(),
			TransactionId: transactionId,
			TraceId:       "",
			Status:        "SUCCESS",
		},
	}

	_ = json.NewEncoder(w).Encode(&response)
}
