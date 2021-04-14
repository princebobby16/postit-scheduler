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
				if ChangeScheduleStatus(connection, schedule, namespace) {
					return
				}

				scheduleChan <- &schedule
			} else {

				logs.Logger.Info("waiting for schedule for %v seconds", schedule.From.Sub(time.Now()))
				/*=======================================	wait till its due before sending  ==============================================*/
				time.Sleep(schedule.From.Sub(time.Now()))
				logs.Logger.Info("Due now")

				if ChangeScheduleStatus(connection, schedule, namespace) {
					return
				}
				scheduleChan <- &schedule
			}
		}
	}
}

func ChangeScheduleStatus(connection *sql.DB, schedule models.PostSchedule, namespace string) bool {
	stmt := fmt.Sprintf("UPDATE %s.schedule SET is_due = $1 WHERE schedule_id = $2;", namespace)
	logs.Logger.Info("Updating the isDue schedule status")
	_, err := connection.Exec(stmt, true, &schedule.ScheduleId)
	if err != nil {
		_ = logs.Logger.Error(err)
		return true
	}
	return false
}

func SchedulePosts(schedules <-chan *models.PostSchedule, postedToFacebook <-chan bool, postedToTwitter chan bool, postedToLinkedIn chan bool, facebookPost chan<- models.SinglePostWithProfiles, twitterPost chan models.SinglePostWithProfiles, linkedInPost chan models.SinglePostWithProfiles, connection *sql.DB, namespace string) {
	// Listen for schedules from the other goroutine
	for s := range schedules {
		logs.Logger.Info("scheduling posts now ... looping through them.")
		for _, postId := range s.PostIds {
			stmt := fmt.Sprintf(`SELECT post_id, post_message, post_images, image_paths, hash_tags FROM %s.post WHERE post_id = $1;`, namespace)
			var post models.Post
			err := connection.QueryRow(stmt, postId).Scan(
				&post.PostId,
				&post.PostMessage,
				pq.Array(&post.PostImages),
				pq.Array(&post.ImagePaths),
				pq.Array(&post.HashTags),
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
					logs.Logger.Info("Facebook Posted status: ", fbStatus, " retrying ...")
					// retry
					facebookPost <- models.SinglePostWithProfiles{
						Post:     post,
						Profiles: s.Profiles,
					}
				}
			}

			if len(s.Profiles.Twitter) != 0 {
				if twStatus := <-postedToTwitter; twStatus == false {
					logs.Logger.Info("Twitter Posted status: ", twStatus, " retrying ...")

					// retry
					twitterPost <- models.SinglePostWithProfiles{
						Post:     post,
						Profiles: s.Profiles,
					}
				}
			}

			if len(s.Profiles.LinkedIn) != 0 {
				if liStatus := <-postedToLinkedIn; liStatus == false {
					logs.Logger.Info("LinkedIn Posted status: ", liStatus, " retrying ...")
					// retry
					linkedInPost <- models.SinglePostWithProfiles{
						Post:     post,
						Profiles: s.Profiles,
					}
				}
			}

			// waiting for the next post
			logs.Logger.Info("waiting for next post for ", s.Duration, " seconds")
			time.Sleep(time.Duration(s.Duration) * time.Second)
		} // for loop s.PostIds
		stmt := fmt.Sprintf("DELETE FROM general_schedule_table WHERE schedule_id = $1")
		_, err := connection.Exec(stmt, s.ScheduleId)
		if err != nil {
			_ = logs.Logger.Error(err)
		}
	} // for loop schedules

	logs.Logger.Info("Schedule done")
}

func SendPostToFacebook(post <-chan models.SinglePostWithProfiles, posted chan<- bool, namespace string, sId string, connection *sql.DB) {
	for p := range post {
		logs.Logger.Info("facebook")
		logs.Logger.Info(p.Post.PostMessage, "\a\t<=======>\t", p.Post.ImagePaths)
		// Post to facebook page
		err := PostToFacebook(p, namespace, connection)
		if err != nil {
			_ = logs.Logger.Error(err)
			posted <- false
		} else {
			stmt := fmt.Sprintf("UPDATE %s.post SET post_fb_status = $1 WHERE post_id = $2;", namespace)
			logs.Logger.Info(p.Post.PostId)
			_, err = connection.Exec(stmt, true, p.Post.PostId)
			if err != nil {
				_ = logs.Logger.Error(err)
				return
			}

			// select the ids from the dbs
			var ids []string
			stmt = fmt.Sprintf("SELECT posted_fb_ids FROM general_schedule_table WHERE schedule_id = $1")
			err = connection.QueryRow(stmt, sId).Scan(pq.Array(&ids))
			if err != nil {
				_ = logs.Logger.Error(err)
				return
			}

			ids = append(ids, p.Post.PostId)

			stmt = fmt.Sprintf("UPDATE general_schedule_table SET posted_fb_idS = $1 WHERE schedule_id = $2")
			_, err = connection.Exec(stmt, pq.Array(&ids), sId)
			if err != nil {
				_ = logs.Logger.Error(err)
				return
			}

			posted <- true
		}
	}
}

func SendPostToTwitter(post <-chan models.SinglePostWithProfiles, posted chan<- bool, namespace string, sId string, connection *sql.DB) {
	for p := range post {
		logs.Logger.Info("twitter")
		logs.Logger.Info(p.Post.PostMessage, "\t<======>\t", p.Post.ImagePaths)

		err := PostToTwitter(p, namespace, connection)
		if err != nil {
			_ = logs.Logger.Error(err)
			posted <- false
		} else {
			stmt := fmt.Sprintf("UPDATE %s.post SET post_tw_status = $1 WHERE post_id = $2;", namespace)
			_, err = connection.Exec(stmt, true, p.Post.PostId)
			if err != nil {
				_ = logs.Logger.Error(err)
				return
			}

			// select the ids from the dbs
			var ids []string
			stmt = fmt.Sprintf("SELECT posted_tw_ids FROM general_schedule_table WHERE schedule_id = $1")
			err = connection.QueryRow(stmt, sId).Scan(pq.Array(&ids))
			if err != nil {
				_ = logs.Logger.Error(err)
				return
			}

			ids = append(ids, p.Post.PostId)

			stmt = fmt.Sprintf("UPDATE general_schedule_table SET posted_tw_idS = $1 WHERE schedule_id = $2")
			_, err = connection.Exec(stmt, pq.Array(&ids), sId)
			if err != nil {
				_ = logs.Logger.Error(err)
				return
			}
			posted <- true
		}
	}
}

func SendPostToLinkedIn(post <-chan models.SinglePostWithProfiles, posted chan<- bool, namespace string, sId string, connection *sql.DB) {
	for p := range post {
		logs.Logger.Info("linkedin")
		logs.Logger.Info(p.Post.PostMessage, "====", p.Post.ImagePaths)

		err := PostToLinkedIn(p, namespace, connection)
		if err != nil {
			logs.Logger.Info(err)
			posted <- false
		} else {
			stmt := fmt.Sprintf("UPDATE %s.post SET post_li_status = $1 WHERE post_id = $2;", namespace)
			_, err = connection.Exec(stmt, true, p.Post.PostId)
			if err != nil {
				_ = logs.Logger.Error(err)
				return
			}

			// select the ids from the dbs
			var ids []string
			stmt = fmt.Sprintf("SELECT posted_li_ids FROM general_schedule_table WHERE schedule_id = $1")
			err = connection.QueryRow(stmt, sId).Scan(pq.Array(&ids))
			if err != nil {
				_ = logs.Logger.Error(err)
				return
			}

			ids = append(ids, p.Post.PostId)

			stmt = fmt.Sprintf("UPDATE general_schedule_table SET posted_li_idS = $1 WHERE schedule_id = $2")
			_, err = connection.Exec(stmt, pq.Array(&ids), sId)
			if err != nil {
				_ = logs.Logger.Error(err)
				return
			}
			posted <- true
		}
	}
}
