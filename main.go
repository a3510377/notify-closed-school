package main

import (
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

func main() {
	main := func() {
		retryCount := 0
		for ; retryCount < 3; retryCount++ {
			if err := checkAndNotification(); err != nil {
				time.Sleep(time.Second * 5) // retry after 5 seconds
				continue
			}
			break
		}
		if retryCount >= 3 {
			log.Println("Retry 3 times, skip check")
		}
	}

	const specTime = "*/15 * * * *"

	c := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(log.Default())))

	c.AddFunc(specTime, main)

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
	for k, v := range data.Data {
		if !areaNamesMap[k] || tmpData[k] == v.State {
			continue
		}
		notifications = append(notifications, v)
		tmpData[k] = v.State
	}

	if len(notifications) > 0 {
		notification(WorkSchoolClose{Date: data.Date, Data: notifications})
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
