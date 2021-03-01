package models

import "time"

type (

	FBPData struct {
		Data []FacebookPageData 		`json:"data"`
	}

	FacebookPageData struct {
		AccessToken string		`json:"access_token"`
		Category string 		`json:"category"`
		CategoryList []Things 	`json:"category_list"`
		Name string				`json:"name"`
		Id string				`json:"id"`
		Tasks []string			`json:"tasks"`
	}

	Things struct {
		Id string			`json:"id"`
		Name string 		`json:"name"`
	}

	FacebookUserData struct {
		UserId string
		AccessToken string
	}

	PostsWithPermission struct {
		ScheduleId string
		Posts []Post
		PostToFeed bool
	}

	SinglePostWithPermission struct {
		ScheduleId string
		Post Post
		PostToFeed bool
	}

	Post struct {
		ScheduleId   string
		PostId       string
		PostMessage  string
		PostImage    []byte
		ImageExtension string
		HashTags     []string
		PostStatus   bool
		PostPriority bool
		CreatedOn    time.Time
		UpdatedOn    time.Time
	}

	PostSchedule struct {
		ScheduleId    string    `json:"schedule_id"`
		ScheduleTitle string    `json:"schedule_title"`
		PostToFeed bool 		`json:"post_to_feed"`
		From          time.Time `json:"from"`
		To            time.Time `json:"to"`
		PostIds       []string  `json:"post_ids"`
		Duration      float64   `json:"duration"`
		CreatedOn     time.Time `json:"created_on"`
		UpdatedOn     time.Time `json:"updated_on"`
	}

	FetchPostResponse struct {
		Data []DbPost `json:"data"`
		Meta Meta     `json:"meta"`
	}

	FetchSchedulePostResponse struct {
		Data []PostSchedule `json:"data"`
		Meta Meta           `json:"meta"`
	}

	StandardResponse struct {
		Data Data `json:"data"`
		Meta Meta `json:"meta"`
	}

	DbPost struct {
		PostId       string    `json:"post_id"`
		PostMessage  string    `json:"post_message"`
		PostImage    string    `json:"post_image"`
		HashTags     []string  `json:"hash_tags"`
		PostStatus   bool      `json:"post_status"`
		PostPriority bool      `json:"post_priority"`
		CreatedOn    time.Time `json:"created_on"`
		UpdatedOn    time.Time `json:"updated_on"`
	}

	Data struct {
		Id        string `json:"id"`
		UiMessage string `json:"ui_message"`
	}

	Meta struct {
		Timestamp     time.Time `json:"timestamp"`
		TransactionId string    `json:"transaction_id"`
		TraceId       string    `json:"trace_id"`
		Status        string    `json:"status"`
	}
)