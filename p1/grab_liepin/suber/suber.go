package suber

import (
	"context"
	"encoding/json"
	"runtime"
	"time"

	"github.com/beiping96/grace"
	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/redis"
	"gitlab.ifchange.com/data/cordwood/util/emoji"
)

const (
	ConstSubQueue = "pyspider_liepin_result_queue"
)

var (
	constWorkDuration = time.Duration(3) * time.Second
)

func Run(ctx context.Context) {
	ticker := time.Tick(constWorkDuration)
	for {
		if err := run(ctx); err != nil {
			log.Warnf("Suber Error %v", err)
		}
		select {
		case <-ticker:
		case <-ctx.Done():
			log.Infof("Suber stopped")
			return
		}
	}
}

func run(ctx context.Context) error {
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		buf := make([]byte, 1<<10)
		num := runtime.Stack(buf, false)
		log.Errorf("Suber PANIC %v %v %v",
			err, num, string(buf))
	}()

	for {
		result, err := sub(ConstSubQueue)
		if err == errEmptySubQueue {
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "Sub")
		}
		grace.Go(func(ctx context.Context) {
			if err := result.handle(); err != nil {
				log.Errorf("Sub Handle %+v %v",
					result, err)
			}
		})
	}
}

var (
	errEmptySubQueue = errors.New("empty sub queue")
)

func sub(subQueue string) (*SubResult, error) {
	result, err := subWorker(subQueue)
	if err != errEmptySubQueue {
		log.Debugf("Sub %s receive %+v %v",
			subQueue, result, err)
	}
	return result, err
}

func subWorker(subQueue string) (*SubResult, error) {
	conn, err := redis.GetConstClient()
	if err != nil {
		return nil, errors.Wrap(err, "Get Redis Const Client")
	}
	defer conn.Close()

	jsonResult, err := conn.DoBytes("RPOP", subQueue)
	if err == conn.ErrNil() {
		return nil, errEmptySubQueue
	}
	if err != nil {
		return nil, errors.Wrap(err, "Redis Cmd RPOP")
	}

	jsonResult = []byte(emoji.Remove(string(jsonResult)))

	result := new(SubResult)
	err = json.Unmarshal(jsonResult, result)
	if err != nil {
		return nil, errors.Wrapf(err, "Json Unmarshal %s", string(jsonResult))
	}

	return result, nil
}
