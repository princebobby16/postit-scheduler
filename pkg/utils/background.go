package utils

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"gitlab.com/pbobby001/postit-scheduler/pkg/logs"
	"gitlab.com/pbobby001/postit-scheduler/pkg/models"
	"time"
)

func HibernateSchedule(connection *sql.DB, schedule models.PostSchedule, namespace string, scheduleChan chan<- *models.PostSchedule) {
	defer close(scheduleChan)
	if schedule.ScheduleId != "" {
		/* Get all schedules that aren't due yet */
		if !schedule.To.Before(time.Now()) {
			// if the schedule is due
			if schedule.From.Before(time.Now()) || schedule.From.Equal(time.Now()) {
				// Do this
				logs.Logger.Info("Schedule is due")

				// update the is_due status in the db
				stmt := fmt.Sprintf("UPDATE %s.schedule SET is_due = $1 WHERE schedule_id = $2;", namespace)
				logs.Logger.Info("Updating the isDue schedule status")
				_, err := connection.Exec(stmt, true, schedule.ScheduleId)
				if err != nil {
					_ = logs.Logger.Error(err)
					return
				}

				scheduleChan <- &schedule
			} else {

				logs.Logger.Info("waiting for schedule for %v seconds", schedule.From.Sub(time.Now()))
				/*=======================================	wait till its due before sending  ==============================================*/
				time.Sleep(schedule.From.Sub(time.Now()))
				logs.Logger.Info("Due now")

				stmt := fmt.Sprintf("UPDATE %s.schedule SET is_due = $1 WHERE schedule_id = $2;", namespace)
				logs.Logger.Info("Updating the isDue schedule status")
				_, err := connection.Exec(stmt, true, schedule.ScheduleId)
				if err != nil {
					_ = logs.Logger.Error(err)
					return
				}

				scheduleChan <- &schedule
			}
		}
	}
}

func SchedulePosts(schedules <-chan *models.PostSchedule, postedToFacebook <-chan bool, postedToTwitter chan bool, postedToLinkedIn chan bool, facebookPost chan<- models.SinglePostWithProfiles, twitterPost chan models.SinglePostWithProfiles, linkedInPost chan models.SinglePostWithProfiles, connection *sql.DB, namespace string) {
	// Listen for posts from the other goroutine
	for s := range schedules {
		logs.Logger.Info("scheduling posts now...\nlooping through them.")
		for _, postId := range s.PostIds {
			stmt := fmt.Sprintf(`SELECT * FROM %s.post WHERE post_id = $1;`, namespace)
			var post models.Post
			err := connection.QueryRow(stmt, postId).Scan(
				&post.PostId,
				&post.FacebookPostId,
				&post.PostMessage,
				pq.Array(&post.PostImages),
				pq.Array(&post.ImagePaths),
				pq.Array(&post.HashTags),
				&post.PostStatus,
				&post.Scheduled,
				&post.PostPriority,
				&post.CreatedOn,
				&post.UpdatedOn,
			)
			if err != nil {
				_ = logs.Logger.Error(err)
				return
			}

			// sending post to social media profiles
			if len(s.Profiles.Facebook) != 0 {
				facebookPost <- models.SinglePostWithProfiles{
					Post:     post,
					Profiles: s.Profiles,
				}
			}

			if len(s.Profiles.Twitter) != 0 {
				twitterPost <- models.SinglePostWithProfiles{
					Post:     post,
					Profiles: s.Profiles,
				}
			}

			if len(s.Profiles.LinkedIn) != 0 {
				linkedInPost <- models.SinglePostWithProfiles{
					Post:     post,
					Profiles: s.Profiles,
				}
			}

			// making sure that the posts have been posted successfully
			if len(s.Profiles.Facebook) != 0 {
				if fbStatus := <-postedToFacebook; fbStatus == false {
					logs.Logger.Info(fbStatus)
					// retry
					facebookPost <- models.SinglePostWithProfiles{
						Post:     post,
						Profiles: s.Profiles,
					}
				}
			}

			if len(s.Profiles.Twitter) != 0 {
				if twStatus := <-postedToTwitter; twStatus == false {
					logs.Logger.Info(twStatus)
					// retry
					twitterPost <- models.SinglePostWithProfiles{
						Post:     post,
						Profiles: s.Profiles,
					}
				}
			}

			if len(s.Profiles.LinkedIn) != 0 {
				if liStatus := <-postedToLinkedIn; liStatus == false {
					logs.Logger.Info(liStatus)
					// retry
					linkedInPost <- models.SinglePostWithProfiles{
						Post:     post,
						Profiles: s.Profiles,
					}
				}
			}

			// waiting for the next post
			logs.Logger.Info("waiting for next post")
			time.Sleep(time.Duration(s.Duration) * time.Second)
		} // for loop s.PostIds
	} // for loop schedules
	logs.Logger.Info("Schedule done")
}

func SendPostToFacebook(post <-chan models.SinglePostWithProfiles, posted chan<- bool, namespace string, connection *sql.DB) {
	for p := range post {
		logs.Logger.Info("facebook")
		logs.Logger.Info(p.Post.PostMessage, "\a\t<=======>\t", p.Post.ImagePaths)
		// Post to facebook page
		err := PostToFacebook(p, namespace, connection)
		if err != nil {
			_ = logs.Logger.Error(err)
			posted <- false
		} else {

			stmt := fmt.Sprintf("UPDATE %s.post SET post_status = $1 WHERE post_id = $2;", namespace)
			_, err = connection.Exec(stmt, true, p.Post.PostId)
			if err != nil {
				_ = logs.Logger.Error(err)
				posted <- true
			}
			posted <- true
		}
	}
}

func SendPostToTwitter(post <-chan models.SinglePostWithProfiles, posted chan<- bool, namespace string, connection *sql.DB) {
	for p := range post {
		logs.Logger.Info("twitter")
		logs.Logger.Info(p.Post.PostMessage, "\t<======>\t", p.Post.ImagePaths)

		err := PostToTwitter(p, namespace, connection)
		if err != nil {
			_ = logs.Logger.Error(err)
			posted <- false
		} else {
			stmt := fmt.Sprintf("UPDATE %s.post SET post_status = $1 WHERE post_id = $2;", namespace)
			_, err = connection.Exec(stmt, true, p.Post.PostId)
			if err != nil {
				_ = logs.Logger.Error(err)
				posted <- true
			}
			posted <- true
		}
	}
}

func SendPostToLinkedIn(post <-chan models.SinglePostWithProfiles, posted chan<- bool, namespace string, connection *sql.DB) {
	for p := range post {
		logs.Logger.Info("linkedin")
		logs.Logger.Info(p.Post.PostMessage, "====", p.Post.ImagePaths)

		err := PostToLinkedIn(p, namespace, connection)
		if err != nil {
			logs.Logger.Info(err)
			posted <- false
		} else {
			stmt := fmt.Sprintf("UPDATE %s.post SET post_status = $1 WHERE post_id = $2;", namespace)
			_, err = connection.Exec(stmt, true, p.Post.PostId)
			if err != nil {
				_ = logs.Logger.Error(err)
				posted <- true
			}
			posted <- true
		}
	}
}
