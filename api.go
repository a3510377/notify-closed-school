package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

const (
	WorkSchoolCloseURL   = "https://www.dgpa.gov.tw/typh/daily/nds.html"
	UA                   = "notifyNotifyClosedSchool (https://github.com/a3510377, 1.0.0) Golang/1.20"
	DiscordMessageAPIUrl = "https://discord.com/api/channels/%d/messages"
	LineMessageAPIUrl    = "https://notify-api.line.me/api/notify"
)

var (
	noClose               = []string{"尚未列入警戒區", "今天照常上班、照常上課", "明天照常上班、照常上課"}
	noClassMap            = map[string]void{}
	timeMatch             = regexp.MustCompile(`\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}`)
	countyCityPrefixRegex = regexp.MustCompile("^.{2}[縣市]")
)

// const WorkSchoolCloseURL = "https://alerts.ncdr.nat.gov.tw/RssAtomFeed.ashx?AlertType=33"
// 由於政府資料開放平臺的資料更新時間不穩定，因此使用 https://www.dgpa.gov.tw/

type WorkSchoolClose struct {
	Date *time.Time
	Data []WorkSchoolCloseData
}
type WorkSchoolCloseMap struct {
	Date *time.Time
	Data map[string]WorkSchoolCloseData
}
type WorkSchoolCloseData struct {
	County  string
	Details []string
}

func init() {
	for _, v := range noClose {
		noClassMap[v] = void{}
	}
}

func GetClosedSchool() (*WorkSchoolCloseMap, error) {
	c := colly.NewCollector()
	result := WorkSchoolCloseMap{Data: map[string]WorkSchoolCloseData{}}

	c.OnHTML("#Content>.Content_Updata>h4:first-child", func(e *colly.HTMLElement) {
		// "更新時間：2023/07/28 11:55:03"
		match := timeMatch.FindStringSubmatch(strings.TrimSpace(e.Text))[0]
		location, _ := time.LoadLocation("Asia/Taipei")
		if date, err := time.ParseInLocation("2006/01/02 15:04:05", match, location); err == nil {
			result.Date = &date
		}
	})
	c.OnHTML("#Table>.Table_Body>tr:not(:last-child)", func(e *colly.HTMLElement) {
		values := e.ChildTexts("td")

		// 無停班停課訊息。
		if lenValues := len(values); lenValues == 1 {
			return
		} else if lenValues > 2 {
			values = values[1:3]
		}

		county, state := strings.TrimSpace(values[0]), strings.TrimSpace(values[1])

		details := []string{}
		for _, v := range regexp.MustCompile("。|\n").Split(state, -1) {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if _, ok := noClassMap[v]; ok {
				continue
			}

			details = append(details, countyCityPrefixRegex.ReplaceAllString(v, ""))
		}

		result.Data[county] = WorkSchoolCloseData{
			County:  county,
			Details: details,
		}
	})

	if err := c.Visit(WorkSchoolCloseURL); err != nil {
		return nil, err
	}

	c.Wait()
	return &result, nil
}

/* ----- notify ----- */
func NotifyLine(values WorkSchoolClose) {
	for _, TOKEN := range ConfigData.Line.Tokens {
		if TOKEN == "" {
			log.Println("Line token is empty")
			continue
		}

		text := "\n"
		for _, v := range values.Data {
			text += fmt.Sprintf("%s: \n  %s\n", v.County, strings.Join(v.Details, "\n  "))
		}
		data := url.Values{"message": {text}}.Encode()
		req, _ := http.NewRequest("POST", LineMessageAPIUrl, strings.NewReader(data))

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Content-Length", strconv.Itoa(len(data)))
		req.Header.Set("Authorization", "Bearer "+TOKEN)
		req.Header.Set("User-Agent", UA)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("Error send Line notification: %s\n", err)
			return
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			data, _ := io.ReadAll(resp.Body)
			log.Printf("Error send Line notification: %s\nResponse: %s\nSend: ", err, data)
			data, _ = io.ReadAll(resp.Request.Body)
			log.Println(string(data))
		}
	}
}

func NotifyDiscord(values WorkSchoolClose) {
	discordConfig := ConfigData.Discord
	TOKEN := discordConfig.TOKEN

	fields := []map[string]any{}
	for _, v := range values.Data {
		fields = append(fields, map[string]any{
			"name":   v.County,
			"value":  strings.Join(v.Details, "\n"),
			"inline": true,
		})
	}
	contentByte, _ := json.Marshal(map[string]any{"embeds": []map[string]any{{
		"title":  "⚠️ 停班停課通知 ⚠️",
		"fields": fields,
		"color":  0xff0000, // red
		"footer": map[string]any{
			"text":      "資料來源: https://www.dgpa.gov.tw/",
			"icon_url":  "https://avatars.githubusercontent.com/u/70706886?v=4",
			"timestamp": values.Date.Unix(),
		},
	}}})
	bodyReader := bytes.NewReader(contentByte)

	if TOKEN == "" {
		log.Println("Discord token is empty")
	} else {
		for _, id := range discordConfig.ChannelIDs {
			// multiple concurrent requests
			go func(data bytes.Reader, id int64) { // id is channel ID
				req, _ := http.NewRequest("POST", fmt.Sprintf(DiscordMessageAPIUrl, id), &data)

				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bot "+TOKEN)
				req.Header.Set("User-Agent", UA)

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Printf("Error send discord: %s\nID: %d\n", err, id)
					return
				}

				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
					data, _ := io.ReadAll(resp.Body)
					log.Printf("Error send discord: %s\nID: %d\nResponse: %s\nSend: ", err, id, data)
					data, _ = io.ReadAll(resp.Request.Body)
					log.Println(string(data))
				}
			}(*bodyReader, id)
		}
	}

	for _, url := range discordConfig.Webhooks {
		// multiple concurrent requests
		go func(data bytes.Reader, url string) {
			req, _ := http.NewRequest("POST", url, &data)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", UA)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Println("Error send discord webhook: ", err, "\nURL:", url)
				return
			}

			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
				data, _ := io.ReadAll(resp.Body)
				log.Printf("Error send discord webhook: %s\nURL: %s\nResponse: %s\n", resp.Status, url, data)
			}
		}(*bodyReader, url)
	}
}
