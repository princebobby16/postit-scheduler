package utils

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/huandu/facebook"
	"gitlab.com/pbobby001/postit-scheduler/pkg/logs"
	"gitlab.com/pbobby001/postit-scheduler/pkg/models"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func PostToFacebook(post models.SinglePostWithProfiles, namespace string, connection *sql.DB) error {
	// make sure user wants to post to fb
	// iterate over the facebook ids
	// to get the access_token stored in the database
	for _, fb := range post.Profiles.Facebook {
		// build the query
		stmt := fmt.Sprintf(
			`SELECT user_id, user_access_token FROM %s.application_info WHERE user_id = $1;`,
			namespace,
		)

		// create an fbUser placeholder to store facebook data
		var fbUser models.FacebookUserData
		logs.Logger.Info(fb)
		// run the query and store the value in the fbUser placeholder
		err := connection.QueryRow(stmt, fb).Scan(
			&fbUser.UserId,
			&fbUser.AccessToken,
		)
		if err != nil {
			_ = logs.Logger.Error(err)
			return err
		}

		logs.Logger.Info("Retrieving page info from facebook")
		// Get a list of pages first
		result, err := facebook.Get("/"+fbUser.UserId+"/accounts",
			facebook.Params{
				"access_token": fbUser.AccessToken,
			},
		)
		if err != nil {
			return err
		}

		// Decode the data into fbPageData object
		var fbPageData models.FBPData
		err = result.Decode(&fbPageData)
		if err != nil {
			return err
		}
		logs.Logger.Info(fbPageData)

		if post.Post.PostImages != nil {
			// send post with image(s)
			resp, err := SendPostWithImageToFacebook(post, namespace, fbPageData)
			if err != nil {
				return err
			}

			err = SetFacebookPostIdColumn(resp, namespace, err, post.Post.PostId, fbUser.UserId, connection)
			if err != nil {
				return err
			}
		} else {
			//send post without image
			resp, err := SendPostWithoutImageToFacebook(post, fbPageData)
			if err != nil {
				return err
			}
			logs.Logger.Info(resp.Get("id"))

			err = SetFacebookPostIdColumn(resp, namespace, err, post.Post.PostId, fbUser.UserId, connection)
			if err != nil {
				return err
			}
		}

	} // for loop post.Profiles.Facebook

	return nil
}

func SetFacebookPostIdColumn(resp facebook.Result, namespace string, err error, id string, userId string, connection *sql.DB) error {
	fbId := fmt.Sprintf("%s", resp.Get("id"))
	if fbId != "" {
		query := fmt.Sprintf(`UPDATE %s.post SET facebook_post_id = $1, facebook_user_id = $2 WHERE post_id = $3;`, namespace)
		_, err = connection.Exec(query, &userId, &fbId, id)
		if err != nil {
			return err
		}
	} else {
		return errors.New("unable to get facebook id")
	}
	return nil
}

func SendPostWithImageToFacebook(post models.SinglePostWithProfiles, namespace string, fbPageData models.FBPData) (facebook.Result, error) {
	var results facebook.Result

	wd, err := os.Getwd()
	if err != nil {
		return facebook.Result{}, err
	}
	imageDir := filepath.Join(wd, "\\pkg\\"+namespace)
	// create a new directory for storing the image
	err = os.Mkdir(imageDir, 0755)
	if err != nil {
		if os.IsExist(err) {
			_ = logs.Logger.Warn(err)
		} else {
			return facebook.Result{}, err
		}
	}

	var imageName string
	var ids []interface{}

	for j := 0; j < len(post.Post.ImagePaths); j++ {
		images := strings.Split(post.Post.ImagePaths[j], "/")
		imageName = images[len(images)-1]
		imageFile, err := os.Create(imageDir + "/" + imageName)
		if err != nil {
			return facebook.Result{}, err
		}

		err = ioutil.WriteFile(imageFile.Name(), post.Post.PostImages[j], os.ModeAppend)
		if err != nil {
			return facebook.Result{}, err
		}

		if fbPageData.Data != nil {
			for _, pageData := range fbPageData.Data {
				var id interface{}
				resp, err := facebook.Post("/"+pageData.Id+"/photos", facebook.Params{
					"published":    false,
					"file":         facebook.File(imageFile.Name()),
					"access_token": pageData.AccessToken,
				})
				if err != nil {
					return facebook.Result{}, err
				}
				logs.Logger.Info(resp.Get("id"))
				logs.Logger.Info(resp.Get("upload_session"))
				logs.Logger.Info(resp.UsageInfo().Page)
				id = resp.Get("id")
				ids = append(ids, id)
			}
		}

	} // for loop post.Post.ImagePaths

	var media []string
	for _, mediaID := range ids {
		media = append(media, fmt.Sprintf("{\"media_fbid\":\"%v\"}", mediaID))
	}

	logs.Logger.Info(media)

	message, err := GeneratePostMessageWithHashTags(post.Post)
	if err != nil {
		return facebook.Result{}, err
	}

	for _, pageData := range fbPageData.Data {
		resp, err := facebook.Post("/"+pageData.Id+"/feed", facebook.Params{
			"access_token":   pageData.AccessToken,
			"attached_media": media,
			"message":        message,
		})
		if err != nil {
			return facebook.Result{}, err
		}

		results = resp
		logs.Logger.Info(resp.UsageInfo().App)
		logs.Logger.Info(resp.UsageInfo().Page)
		logs.Logger.Info("Post Id: ", resp.Get("id"))
	}

	return results, nil
}

func SendPostWithoutImageToFacebook(post models.SinglePostWithProfiles, fbPageData models.FBPData) (facebook.Result, error) {
	var results facebook.Result
	for _, pageData := range fbPageData.Data {
		message, err := GeneratePostMessageWithHashTags(post.Post)
		if err != nil {
			return facebook.Result{}, err
		}

		resp, err := facebook.Post("/"+pageData.Id+"/feed", facebook.Params{
			"access_token": pageData.AccessToken,
			"message":      message,
		})
		if err != nil {
			return facebook.Result{}, err
		}

		results = resp
		logs.Logger.Infof("PostID: %s", resp.Get("id"))
	}
	return results, nil
}
