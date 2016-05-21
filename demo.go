package main

import (
	"log"
	"strconv"
	"strings"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
	"gopkg.in/redis.v3"
)

var config GlobalConfig

type FailLog struct {
	StatusCode int
	Token      string
}

func newPushClient(client *apns2.Client, c chan string, logChan chan FailLog) {
	var token, payload string
	var notificationArgs []string
	notification := apns2.Notification{Topic: config.Topic}
	for {
		notificationArgs = strings.SplitN(<-c, " ", 2)
		if len(notificationArgs) != 2 {
			log.Println("invalid format message")
		} else {
			token = notificationArgs[0]
			payload = notificationArgs[1]

			notification.DeviceToken = token
			notification.Payload = payload
			res, err := client.Push(&notification)

			if err != nil {
				log.Printf("Error: ", err)
			} else {
				if res.StatusCode != 200 {
					logChan <- FailLog{
						Token:      token,
						StatusCode: res.StatusCode,
					}
				}
				log.Printf("%v-%v: '%v'\n", res.StatusCode, token, res.Reason)
			}
		}
	}
}

func logIntoRedis(c chan FailLog) {
	redisClient := newRedisClient()

	for {
		failLog := <-c
		_, err := redisClient.SAdd(config.RedisListKey+":"+strconv.Itoa(failLog.StatusCode), failLog.Token).Result()
		if err != nil {
			log.Println(err)
			redisClient.Close()
			//reconnect redis
			redisClient = newRedisClient()
		}
	}
}

func newRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:        config.RedisAddr,
		Password:    config.RedisPassword,
		DB:          config.RedisDB,
		ReadTimeout: 0,
	})
}

func init() {
	config = getConfig()
}

func main() {
	cert, pemErr := certificate.FromPemFile(config.CertificatePath, "")
	if pemErr != nil {
		log.Fatalf("Error retrieving certificate `%v`: %v", config.CertificatePath, pemErr)
	}
	logChans := make(chan FailLog, config.RoutineCount)
	chans := make(chan string, config.RoutineCount)
	client := apns2.NewClient(cert)
	if config.Mode == "development" {
		client.Development()
	} else {
		client.Production()
	}

	//push goroutine
	for i := uint(0); i < config.RoutineCount; i++ {
		go newPushClient(client, chans, logChans)
	}

	//redis log goroutine
	go logIntoRedis(logChans)

	redisClient := newRedisClient()

	for {
		message, err := redisClient.BLPop(0, config.RedisListKey).Result()
		if err != nil {
			log.Println(err)
			redisClient.Close()
			//reconnect redis
			redisClient = newRedisClient()
		}
		log.Println(message[1])
		chans <- message[1]
	}
}
