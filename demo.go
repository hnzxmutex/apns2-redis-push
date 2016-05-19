package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/go-ini/ini"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
	"gopkg.in/redis.v3"
)

type Config struct {
	CertificatePath                        string
	RedisAddr, RedisPassword, RedisListKey string
	RoutineCount                           uint
	Mode, Topic                            string
}

var config Config

//get config
func init() {
	var configPath string
	flag.StringVar(&configPath, "config-path", "./config.ini", "the config.ini path")
	flag.Parse()

	ini.MapTo(&config, configPath)
	fmt.Println(config)
}

func newPushClient(client *apns2.Client, c chan string) {
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
				fmt.Printf("%v: '%v'\n", res.StatusCode, res.Reason)
			}
		}
	}
}

func newRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:        config.RedisAddr,
		Password:    config.RedisPassword,
		DB:          0,
		ReadTimeout: 0,
	})
}

func main() {
	cert, pemErr := certificate.FromPemFile(config.CertificatePath, "")
	if pemErr != nil {
		log.Fatalf("Error retrieving certificate `%v`: %v", config.CertificatePath, pemErr)
	}
	chans := make(chan string, config.RoutineCount)

	for i := uint(0); i < config.RoutineCount; i++ {
		client := apns2.NewClient(cert)
		if config.Mode == "development" {
			client.Development()
		} else {
			client.Production()
		}
		go newPushClient(client, chans)
	}

	redisClient := newRedisClient()

	for {
		message, err := redisClient.BLPop(0, config.RedisListKey).Result()
		if err != nil {
			log.Println(err)
			redisClient.Close()
			redisClient = newRedisClient()
		}
		fmt.Println(message[1])
		chans <- message[1]
	}
}
