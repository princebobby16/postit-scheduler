package utils

import (
	"database/sql"
	"gitlab.com/pbobby001/postit-scheduler/pkg/models"
)

//goland:noinspection GoUnusedParameter,GoUnusedParameter,GoUnusedParameter
func PostToTwitter(p models.SinglePostWithProfiles, namespace string, connection *sql.DB) error {

	return nil
}

//goland:noinspection GoUnusedParameter,GoUnusedParameter,GoUnusedParameter
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
