package bq

import (
	"context"
	"fmt"
	"reflect"

	"cloud.google.com/go/bigquery"
	"github.com/democracy-tools/countmein/internal/env"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const (
	EnvKeyBQToken = "BIGQUERY_KEY"

	TableAnnouncement = "announcement"
)

type Client interface {
	Insert(tableId string, src interface{}) error
	GetAnnouncementCount(from int64) (int64, error)
}

type ClientWrapper struct {
	bqClient *bigquery.Client
	dataset  string
}

func NewClientWrapper(gcpProjectID string) Client {

	if key := env.GetEnvSensitive(EnvKeyBQToken); key != "" {
		conf, err := google.JWTConfigFromJSON([]byte(key), bigquery.Scope)
		if err != nil {
			log.Fatalf("failed to config bigquery JWT with %q", err)
		}

		ctx := context.Background()
		client, err := bigquery.NewClient(ctx, gcpProjectID, option.WithTokenSource(conf.TokenSource(ctx)))
		if err != nil {
			log.Fatalf("failed to create bigquery client with %q", err)
		}

		return newClientWrapper(client)
	}

	client, err := bigquery.NewClient(context.Background(), gcpProjectID)
	if err != nil {
		log.Fatalf("failed to create bigquery client without token with %q", err)
	}

	return newClientWrapper(client)
}

func newClientWrapper(client *bigquery.Client) Client {

	return &ClientWrapper{bqClient: client, dataset: env.GetBQDataset()}
}

func (c *ClientWrapper) Insert(tableId string, src interface{}) error {

	items, err := ToInterfaceSlice(src)
	if err != nil {
		return err
	}

	count := len(items)
	start, end := 0, 99
	for {
		if end > count {
			end = count
		}
		err := c.bqClient.Dataset(c.dataset).Table(tableId).Inserter().Put(context.Background(), items[start:end])
		if err != nil {
			log.Errorf("failed to persist '%s.%s' with %q", c.dataset, tableId, err)
			return err
		}
		log.Debugf("inserted '%d:%d' into '%s.%s'", start, end, c.dataset, tableId)
		if end == count {
			break
		}
		start += 100
		end += 100
	}

	return nil
}

func (c *ClientWrapper) GetAnnouncementCount(from int64) (int64, error) {

	query := c.bqClient.Query(`SELECT count(DISTINCT user_device_id) FROM ` + getTableFullName(c.dataset, TableAnnouncement) + ` WHERE user_time > @time AND user_id != 'Simulator'`)
	query.Parameters = []bigquery.QueryParameter{{
		Name:  "time",
		Value: from,
	}}

	iterator, err := query.Read(context.Background())
	if err != nil {
		log.Errorf("failed to execute get announcement count query from time '%d' with '%v'", from, err)
		return -1, err
	}

	var values []bigquery.Value
	err = iterator.Next(&values)
	if err != nil {
		log.Errorf("failed to get announcement count from time '%d' with '%v'", from, err)
		return -1, err
	}
	if len(values) != 1 {
		log.Errorf("unexpected bigquery value '%v' when getting announcement count", values)
		return -1, err
	}
	count, ok := values[0].(int64)
	if !ok {
		log.Errorf("unexpected bigquery value of announcement count '%s'", reflect.TypeOf(values[0]).String())
		return -1, err
	}

	return count, nil
}

func getTableFullName(dataset string, table string) string {

	return fmt.Sprintf("%s.%s.%s", env.GetProjectId(), dataset, table)
}
