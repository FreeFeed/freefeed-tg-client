package frf

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/davidmz/mustbe"
	"github.com/gofrs/uuid"
)

const APITimeout = 10 * time.Second

// API is the FreeFeed API client
type API struct {
	HostName    string
	AccessToken string
	UserAgent   string
}

// GetMe returns the basic information about the current user
func (a *API) GetMe() (*User, error) {
	resp := &struct {
		User *User `json:"users"`
	}{}
	err := a.request("GET", "/v1/users/me", nil, resp)
	return resp.User, err
}

func (a *API) GetEvents() ([]*Event, error) {
	resp := &struct {
		Events []*Event `json:"Notifications"`
		Groups []*User
		Users  []*User
	}{}
	if err := a.request("GET", "/v2/notifications", nil, resp); err != nil {
		return nil, err
	}

	accById := make(map[uuid.UUID]*User)
	for _, a := range resp.Users {
		accById[a.ID] = a
	}
	for _, a := range resp.Groups {
		accById[a.ID] = a
	}

	for _, e := range resp.Events {
		e.AffectedUser = accById[e.AffectedUserID]
		e.CreatedUser = accById[e.CreatedUserID]
		e.Group = accById[e.GroupID]
		e.PostAuthor = accById[e.PostAuthorID]
	}
	return resp.Events, nil
}

func (a *API) GetPost(postID uuid.UUID) (*Post, error) {
	resp := &struct {
		Posts struct {
			Post
			PostedTo []uuid.UUID
		}
		TargetFeeds []Feed `json:"subscriptions"`
		Comments    []Comment
	}{}
	err := a.request("GET", "/v2/posts/"+postID.String()+"?maxComments=all", nil, resp)
	if err == nil {
		for _, feedID := range resp.Posts.PostedTo {
			for _, feed := range resp.TargetFeeds {
				if feed.ID == feedID {
					resp.Posts.Recipients = append(resp.Posts.Recipients, feed)
				}
			}
		}
		resp.Posts.Post.Comments = resp.Comments
	}
	return &resp.Posts.Post, err
}

func (a *API) AcceptSubscriptionRequest(userName string) error {
	return a.request("POST", "/v1/users/acceptRequest/"+userName, &struct{}{}, nil)
}

func (a *API) RejectSubscriptionRequest(userName string) error {
	return a.request("POST", "/v1/users/rejectRequest/"+userName, &struct{}{}, nil)
}

func (a *API) AcceptGroupSubscriptionRequest(userName string, groupName string) error {
	return a.request("POST", "/v1/groups/"+groupName+"/acceptRequest/"+userName, &struct{}{}, nil)
}

func (a *API) RejectGroupSubscriptionRequest(userName string, groupName string) error {
	return a.request("POST", "/v1/groups/"+groupName+"/rejectRequest/"+userName, &struct{}{}, nil)
}

func (a *API) AddComment(postID uuid.UUID, text string) (*Comment, error) {
	resp := &struct {
		Comment *Comment `json:"comments"`
	}{}
	err := a.request("POST", "/v1/comments", newAddCommentRequest(postID, text), resp)
	return resp.Comment, err
}

////

func (a *API) request(method string, uri string, reqObj interface{}, respObj interface{}) (err error) {
	defer mustbe.CatchedAs(&err)

	url := "https://" + a.HostName + uri

	var body io.Reader
	if reqObj != nil {
		bodyBytes := mustbe.OKVal(json.Marshal(reqObj)).([]byte)
		body = bytes.NewBuffer(bodyBytes)
	}

	ctx, cancel := context.WithTimeout(context.Background(), APITimeout)
	defer cancel()

	req := mustbe.OKVal(http.NewRequestWithContext(ctx, method, url, body)).(*http.Request)

	if a.AccessToken != "" {
		req.Header.Add("Authorization", "Bearer "+a.AccessToken)
	}
	if body != nil {
		req.Header.Add("Content-Type", "application/json; charset=utf-8")
	}
	if a.UserAgent != "" {
		req.Header.Set("User-Agent", a.UserAgent)
	}

	resp := mustbe.OKVal(http.DefaultClient.Do(req)).(*http.Response)
	defer resp.Body.Close()

	mustbe.OK(errorFromResponse(resp))

	if respObj != nil {
		data := mustbe.OKVal(ioutil.ReadAll(resp.Body)).([]byte)
		mustbe.OK(json.Unmarshal(data, respObj))
	} else {
		mustbe.OKVal(io.Copy(ioutil.Discard, resp.Body))
	}

	return
}
