package utils

import (
	"database/sql"
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
		// run the query and store the value in the fbUser placeholder
		err := connection.QueryRow(stmt, fb).Scan(
			&fbUser.UserId,
			&fbUser.AccessToken,
		)
		if err != nil {
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

		// send post with image(s)
		if post.Post.PostImages != nil {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			imageDir := filepath.Join(wd, "\\pkg\\"+namespace)
			// create a new directory for storing the image
			err = os.Mkdir(imageDir, 0755)
			if err != nil {
				if os.IsExist(err) {
					_ = logs.Logger.Warn(err)
				} else {
					return err
				}
			}

			var imageName string
			var ids []interface{}

			for j := 0; j < len(post.Post.ImagePaths); j++ {
				images := strings.Split(post.Post.ImagePaths[j], "\\")
				imageName = images[len(images)-1]
				imageFile, err := os.Create(imageDir + "\\" + imageName)
				if err != nil {
					return err
				}

				err = ioutil.WriteFile(imageFile.Name(), post.Post.PostImages[j], os.ModeAppend)
				if err != nil {
					return err
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
							return err
						}
						logs.Logger.Info(resp.Get("id"))
						logs.Logger.Info(resp.Get("upload_session"))
						logs.Logger.Info(resp.UsageInfo().Page)
						id = resp.Get("id")
						ids = append(ids, id)
					}
				}

				err = os.RemoveAll(imageDir)
				if err != nil {
					return err
				}
			} // for loop post.Post.ImagePaths

			var media []string
			for _, mediaID := range ids {
				media = append(media, fmt.Sprintf("{\"media_fbid\":\"%v\"}", mediaID))
			}

			logs.Logger.Info(media)

			message, err := GeneratePostMessageWithHashTags(post.Post)
			if err != nil {
				return err
			}

			for _, pageData := range fbPageData.Data {
				resp, err := facebook.Post("/"+pageData.Id+"/feed", facebook.Params{
					"access_token":   pageData.AccessToken,
					"attached_media": media,
					"message":        message,
				})
				if err != nil {
					return err
				}
				logs.Logger.Info(resp.UsageInfo().App)
				logs.Logger.Info(resp.UsageInfo().Page)
				logs.Logger.Info("Post Id: ", resp.Get("id"))
			}
		}
		//send post without image
		for _, pageData := range fbPageData.Data {
			message, err := GeneratePostMessageWithHashTags(post.Post)
			if err != nil {
				return err
			}

			resp, err := facebook.Post("/"+pageData.Id+"/feed", facebook.Params{
				"access_token": pageData.AccessToken,
				"message":      message,
			})
			logs.Logger.Info(resp.Get("id"))
		}

	} // for loop post.Profiles.Facebook

	return nil
}

func PostToTwitter(p models.SinglePostWithProfiles, namespace string, connection *sql.DB) error {

	return nil
}

func PostToLinkedIn(p models.SinglePostWithProfiles, namespace string, connection *sql.DB) error {

	return nil
}

func GeneratePostMessageWithHashTags(post models.Post) (string, error) {
	m := ""

	if post.HashTags == nil {
		return post.PostMessage, nil
	}

	for i := 0; i < len(post.HashTags); i++ {

		if i == 0 {
			m = post.PostMessage + "\n\n" + post.HashTags[i]
		} else {
			m += "\n" + post.HashTags[i]
		}

	}

	return m, nil
}

/*func Page(post models.Post, token string, id string, nmsp string) (facebook.Result, error) {

	postMessage, err := GeneratePostMessageWithHashTags(post)
	if err != nil {
		return nil, err
	}
	logs.Logger.Info(postMessage)

	logs.Logger.Info("Retrieving page info from facebook")
	// Get a list of pages first
	result, err := facebook.Get("/"+id+"/accounts",
		facebook.Params{
			"access_token": token,
		},
	)
	if err != nil {
		return nil, err
	}

	// Decode the data into fbPageData object
	var fbPageData models.FBPData
	err = result.Decode(&fbPageData)
	if err != nil {
		return nil, err
	}

	logs.Logger.Info(fbPageData)

	if fbPageData.Data != nil {
		for _, d := range fbPageData.Data {
			if post.PostImages == nil {
				logs.Logger.Info("Posting Without Image")
				_res, err := facebook.Post("/"+d.Id+"/feed", facebook.Params{
					"message":      postMessage,
					"access_token": d.AccessToken,
				})
				if err != nil {
					logs.Logger.Error(err)
					return nil, err
				}
				logs.Logger.Info("Posted: ", _res.Get("id"))

			} else if post.PostImages != nil {
				logs.Logger.Info("Posting With Image")
				logs.Logger.Info("image Extension: ", post.ImagePaths)

				for i := 0; i < len(post.PostImages); i++ {
					wd, err := os.Getwd()
					if err != nil {
						_ = logs.Logger.Error(err)
						return nil, err
					}

					// join the working directory path with the path for image storage
					join := filepath.Join(wd, "pkg/"+nmsp)

					// create a new directory for storing the image
					err = os.Mkdir(join, 0755)
					if err != nil {
						if os.IsExist(err) {
							_ = logs.Logger.Warn(err)
						} else {
							//_ = logs.Logger.Error(err)
							return nil, err
						}
					}
				}

				//var completeImagePath string
				//var paths []string
				//for i, e := range post.ImagePaths {
				//	logs.Logger.Info("Creating image file")
				//	blob, err := os.Create(post.PostId + "." + e)
				//	if err != nil {
				//		return nil, err
				//	}
				//
				//	logs.Logger.Info("Writing image content to file")
				//	err = ioutil.WriteFile(blob.Name(), post.PostImages[i], os.ModeAppend)
				//	if err != nil {
				//		return nil, err
				//	}
				//	logs.Logger.Info(blob.Name())
				//
				//	logs.Logger.Info("Getting image full path")
				//	wd, err := os.Getwd()
				//	if err != nil {
				//		return nil, err
				//	}
				//
				//	completeImagePath = filepath.Join(wd, blob.Name())
				//	logs.Logger.Info(completeImagePath)
				//	paths = append(paths, completeImagePath)
				//
				//
				//	//resp, err := facebook.Post("/" + d.Id + "/photos", facebook.Params {
				//	//	"published":      false,
				//	//	"file": facebook.File(completeImagePath),
				//	//	"access_token": d.AccessToken,
				//	//})
				//
				//
				//}
				//_res, err := facebook.Post("/" + d.Id + "/photos", facebook.Params {
				//	"message":      postMessage,
				//	"file": facebook.File(),
				//	"access_token": d.AccessToken,
				//})
				//if err != nil {
				//	logs.Logger.Error(err)
				//	return err
				//}
				//logs.Logger.Info("Posted: ", _res.Get("id"))
				//for _, path := range paths {
				//	// Delete file
				//	logs.Logger.Info("Deleting Image File From Directory")
				//	err = os.Remove(path)
				//	if err != nil {
				//		return nil, err
				//	}
				//}
				log.Println("@done")
			}
		}
	} else {
		logs.Logger.Warn("No Facebook Pages Found")
		return nil, errors.New("no facebook pages found")
	}

	return nil, nil
}*/
