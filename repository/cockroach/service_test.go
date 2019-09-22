// +build integration

package cockroach

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os/exec"
	"testing"
	"time"

	"github.com/jozuenoon/dunder/model"

	"github.com/jozuenoon/dunder/repository"

	"github.com/stretchr/testify/assert"
)

func createDb(database string) error {
	return executeDb("createdb", database)
}

func executeDb(command, database string) error {
	cmd := exec.Command(command, "-p", "26257", "-h", "127.0.0.1", "-U", "root", "-e", database)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func dropDb(database string) error {
	return executeDb("dropdb", database)
}

func extractTagText(tags []*repository.Hashtag) []string {
	var t []string
	for _, h := range tags {
		tt := *h.Text
		t = append(t, tt)
	}
	return t
}

func TestSimpleInsertAndGet(t *testing.T) {
	database := fmt.Sprintf("test_%d", rand.Intn(1000))
	t.Log("using database: ", database)
	assert.NoError(t, createDb(database), "failed to create database")
	defer dropDb(database)
	user := "root"

	svc, err := New(&Config{
		Host:          "localhost",
		ShouldMigrate: true,
		Debug:         false,
		Database:      &database,
		User:          &user,
	})
	assert.NoError(t, err, "failed to create database")
	msg := &repository.CreateMessageRequest{
		UserName: "john@example.com",
		Text:     "my dummy text 1",
		Hashtags: []string{"atwork", "someother"},
	}
	var msgUlid string
	t.Run("create message", func(t *testing.T) {
		msgUlid, err = svc.CreateMessage(context.Background(), msg)
		assert.NoError(t, err, "failed to create message")
	})

	t.Run("get message", func(t *testing.T) {
		rmsg, err := svc.Message(context.Background(), msgUlid)
		assert.NoError(t, err, "failed to get message")
		assert.Equal(t, msg.Text, rmsg.Text, "message text don't match")
		assert.Equal(t, msg.UserName, *rmsg.User.Name, "user name does not match")
		assert.ElementsMatch(t, msg.Hashtags, extractTagText(rmsg.Hashtags), "tags does not match")
	})

}

func TestService_CreateMessage_Message(t *testing.T) {
	database := fmt.Sprintf("test_%d", rand.Intn(1000))
	t.Log("using database: ", database)
	assert.NoError(t, createDb(database), "failed to create database")
	defer dropDb(database)
	user := "root"

	svc, err := New(&Config{
		Host:          "localhost",
		ShouldMigrate: true,
		Debug:         true,
		Database:      &database,
		User:          &user,
	})
	assert.NoError(t, err, "failed to create database")
	tests := []struct {
		name string
		req  *repository.CreateMessageRequest
	}{
		{
			name: "insert1",
			req: &repository.CreateMessageRequest{
				UserName: "john@example.com",
				Text:     "my dummy text 1",
				Hashtags: []string{"atwork", "someother"},
			},
		},
		{
			name: "insert2",
			req: &repository.CreateMessageRequest{
				UserName: "ala@example.com",
				Text:     "my dummy text 2",
				Hashtags: []string{"drift", "carbon"},
			},
		},
		{
			name: "insert3",
			req: &repository.CreateMessageRequest{
				UserName: "john@example.com",
				Text:     "my dummy text 3",
				Hashtags: []string{"grip", "programming"},
			},
		},
		{
			name: "insert4",
			req: &repository.CreateMessageRequest{
				UserName: "cook@example.com",
				Text:     "my dummy text 4",
				Hashtags: []string{"chocolate", "sunflower"},
			},
		},
		{
			name: "insert5",
			req: &repository.CreateMessageRequest{
				UserName: "grimm@example.com",
				Text:     "my dummy text 5",
				Hashtags: []string{"ghost", "oldlady"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mulid, err := svc.CreateMessage(context.Background(), tt.req)
			if err != nil {
				t.Fatal(err)
			}
			rmsg, err := svc.Message(context.Background(), mulid)
			if err != nil {
				t.Fatal(err)
			}
			assert.NoError(t, err, "failed to get message")
			assert.Equal(t, tt.req.Text, rmsg.Text, "message text don't match")
			assert.Equal(t, tt.req.UserName, *rmsg.User.Name, "user name does not match")
			assert.ElementsMatch(t, tt.req.Hashtags, extractTagText(rmsg.Hashtags), "tags does not match")
		})
	}
}

func TestSimpleFilter(t *testing.T) {
	database := fmt.Sprintf("test_%d", rand.Intn(1000))
	t.Log("using database: ", database)
	assert.NoError(t, createDb(database), "failed to create database")
	defer dropDb(database)
	user := "root"

	svc, err := New(&Config{
		Host:          "localhost",
		ShouldMigrate: true,
		Debug:         false,
		Database:      &database,
		User:          &user,
	})
	assert.NoError(t, err, "failed to create database")

	for _, m := range messages {
		_, err := svc.CreateMessage(context.Background(), m)
		assert.NoError(t, err, "failed to create message")
	}

	t.Run("search for first message by tag", func(t *testing.T) {
		filter := &repository.FilterImpl{
			QueryRequest: model.QueryRequest{
				Rules: model.QueryRules{
					Hashtag: "atwork",
				},
			},
		}
		msg := messages[0]
		lrmsg, err := svc.Messages(context.Background(), filter)
		assert.Len(t, lrmsg, 1, "response not equal 1")
		rmsg := lrmsg[0]
		assert.NoError(t, err, "failed to get message")
		assertRequest(t, msg, rmsg)
	})

	t.Run("search for first message by username", func(t *testing.T) {
		filter := &repository.FilterImpl{
			QueryRequest: model.QueryRequest{
				Rules: model.QueryRules{
					UserName: "john@example.com",
				},
			},
		}
		msg := messages[0]
		lrmsg, err := svc.Messages(context.Background(), filter)
		assert.Len(t, lrmsg, 1, "response not equal 1")
		rmsg := lrmsg[0]
		assert.NoError(t, err, "failed to get message")
		assertRequest(t, msg, rmsg)
	})

	t.Run("search for messages in recent time range and follow up with cursor", func(t *testing.T) {
		fromTime := time.Now().Add(-time.Second * 3)
		filter := &repository.FilterImpl{
			QueryRequest: model.QueryRequest{
				FromDate: fromTime,
				ToDate:   fromTime.Add(time.Second * 6),
				Limit:    1,
			},
		}
		// Should return last message
		msg := messages[len(messages)-1]
		lrmsg, err := svc.Messages(context.Background(), filter)
		assert.Len(t, lrmsg, 1, "response not equal 1")
		rmsg := lrmsg[0]
		assert.NoError(t, err, "failed to get message")
		assertRequest(t, msg, rmsg)

		// Search with cursor
		filter = &repository.FilterImpl{
			QueryRequest: model.QueryRequest{
				Cursor: *rmsg.Ulid,
				Limit:  1,
			},
		}
		// Should return one before last message
		msg = messages[len(messages)-2]
		lrmsg, err = svc.Messages(context.Background(), filter)
		assert.Len(t, lrmsg, 1, "response not equal 1")
		rmsg = lrmsg[0]
		assert.NoError(t, err, "failed to get message")
		assertRequest(t, msg, rmsg)
	})
}

func assertRequest(t *testing.T, request *repository.CreateMessageRequest, resp *repository.Message) {
	t.Helper()
	assert.Equal(t, request.Text, resp.Text, "message text don't match")
	assert.Equal(t, request.UserName, *resp.User.Name, "user name does not match")
	assert.ElementsMatch(t, request.Hashtags, extractTagText(resp.Hashtags), "tags does not match")
}

var messages = []*repository.CreateMessageRequest{
	{
		UserName: "john@example.com",
		Text:     "my dummy text 1",
		Hashtags: []string{"atwork", "someother"},
	},
	{
		UserName: "grimma@example.com",
		Text:     "my dummy text 2",
		Hashtags: []string{"great", "work"},
	},
	{
		UserName: "othello@example.com",
		Text:     "my dummy text 3",
		Hashtags: []string{"marble", "milk"},
	},
}

func TestSimpleTrends(t *testing.T) {
	database := fmt.Sprintf("test_%d", rand.Intn(1000))
	//t.Log("using database: ", database)
	//assert.NoError(t, createDb(database), "failed to create database")
	//defer dropDb(database)
	user := "root"

	svc, err := New(&Config{
		Host:          "localhost",
		ShouldMigrate: true,
		Debug:         true,
		Database:      &database,
		User:          &user,
	})
	assert.NoError(t, err, "failed to create database")

	// Put some messages
	for _, m := range messages {
		_, err := svc.CreateMessage(context.Background(), m)
		assert.NoError(t, err, "failed to create message")
	}
	tn := time.Now()
	resp, err := svc.Trends(context.Background(), &repository.FilterImpl{QueryRequest: model.QueryRequest{
		FromDate: tn.Add(-time.Minute * 20),
		ToDate:   tn.Add(time.Minute * 5),
		Rules: model.QueryRules{
			Aggregation: time.Minute,
			Hashtag:     "marble",
		},
	}})
	assert.NoError(t, err, "failed to read trends")

	for _, r := range resp.Trends {
		fmt.Println(r.FromDate, r.ToDate, r.Count)
	}
}
