package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

func main() {
	const specTime = "*/15 * * * *"
	// everyMinute, _ := cron.ParseStandard("* * * * *")

	c := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(log.Default())))

	loop, _ := c.AddFunc(specTime, func() {
		retryCount := 0
		for ; retryCount < 3; retryCount++ {
			log.Println("Check and notification")
			if err := checkAndNotification(); err != nil {
				log.Println("checkAndNotification error", err)
				time.Sleep(time.Second * 5) // retry after 5 seconds
				continue
			}

			log.Println(strings.Repeat("-", 70))
			break
		}
		if retryCount >= 3 {
			log.Println("Retry 3 times, skip check")
		}
	})
	entry := c.Entry(loop)

	go entry.Job.Run() // first run

	// oldSchedule := entry.Schedule
	// entry.Schedule = everyMinute
	// entry.Next = time.Now().In(c.Location())

	c.Run() // loop start
}

func checkAndNotification() error {
	data, err := GetClosedSchool()
	if err != nil {
		log.Println("GetClosedSchool error", err)
		return err
	}

	areaNamesMap := map[string]bool{}
	notifications := []WorkSchoolCloseData{}
	for _, v := range ConfigData.AreaNames {
		areaNamesMap[v] = true
	}

	tmpData := GetTmpDate()
	for county, v := range data.Data {
		countyTmpData, countyExists := tmpData[county]
		if !countyExists {
			tmpData[county] = map[string]bool{}
		}

		details := []string{}
		for _, detail := range v.Details {
			absoluteDetail := ConvertRelativeToAbsoluteTime(detail, *data.Date)
			if _, ok := countyTmpData[absoluteDetail]; ok {
				continue
			}

			details = append(details, detail)
			tmpData[county][absoluteDetail] = true
		}

		notifications = append(notifications, WorkSchoolCloseData{
			County:  county,
			Details: details,
		})
	}

	if len(notifications) > 0 {
		text := ""
		sendNotifications := []WorkSchoolCloseData{}
		for _, v := range notifications {
			if len(v.Details) == 0 {
				continue
			}

			text += fmt.Sprintf("%s: \n  %s\n", v.County, strings.Join(v.Details, "\n  "))

			if !areaNamesMap[v.County] {
				continue
			}

			sendNotifications = append(sendNotifications, v)
		}

		if text != "" {
			fmt.Println(text)
		}

		if len(sendNotifications) > 0 {
			notification(WorkSchoolClose{Date: data.Date, Data: sendNotifications})
		}
	}

	// clear timeout data
	nowTime := time.Now()
	for _, data := range tmpData {
		for k := range data {
			if HasStatusIsOld(k, nowTime) {
				delete(data, k)
			}
		}
	}

	WriteTmpDate(tmpData)
	return nil
}

func notification(notifications WorkSchoolClose) {
	if ConfigData.Discord.Enable {
		go NotifyDiscord(notifications)
	}

	if ConfigData.Line.Enable {
		go NotifyLine(notifications)
	}
}
