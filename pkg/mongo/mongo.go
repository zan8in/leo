package mongo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MONGO struct {
	host    string
	port    string
	retries int
	timeout int
}

var (
	ErrRtries          = errors.New("retries exceeded")
	ErrNoHost          = errors.New("no input host provided")
	default_mongo_port = "27017"

	ErrLoginFailed = "Login failed for user"
)

func New(host, port string, retries, timeout int) (*MONGO, error) {
	if len(host) == 0 {
		return nil, ErrNoHost
	}

	if len(port) == 0 {
		port = default_mongo_port
	}

	mongo := &MONGO{host: host, port: port, retries: retries, timeout: timeout}
	mongo.retries = 1

	return mongo, nil
}

func (mongo *MONGO) AuthRetries(user, password string) (err error) {
	sum := 0
	for {
		if sum > mongo.retries {
			return errors.New(ErrRtries.Error() + ", " + err.Error())
		}

		err = mongo.auth(user, password)
		if err != nil && strings.Contains(err.Error(), ErrLoginFailed) {
			return err
		}
		if err != nil {
			sum++
			time.Sleep(500 * time.Millisecond)
			continue
		}

		return nil
	}
}

func (mongo *MONGO) auth(user, password string) error {

	cred := qmgo.Credential{
		Username: user,
		Password: password,
	}

	conf := qmgo.Config{
		Uri:      fmt.Sprintf("mongodb://%s:%s", mongo.host, mongo.port),
		Database: "admin",
		Coll:     fmt.Sprint("image", time.Now().Format("20060102")),
		Auth:     &cred,
	}

	ctx := context.Background()
	cli, err := qmgo.Open(ctx, &conf)
	if err != nil {
		return err
	}

	err = cli.Ping(3)
	if err != nil {
		return err
	}

	return err
}

func (mo *MONGO) authNoPass() error {
	clientOptions := options.Client().ApplyURI("mongodb://" + mo.host + ":" + mo.port)

	mgo, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return err
	}

	err = mgo.Ping(context.Background(), nil)
	if err != nil {
		return err
	}

	return err
}
