package function

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/spf13/cast"
)

const (
	botToken        = "6475922354:AAFu0BgS5HZQthcqCuE1Dkne2VZsqoBsiL0"
	chatID          = int64(-4087622098)
	baseUrl         = "https://api.admin.u-code.io"
	logFunctionName = "ucode-template"
	IsHTTP          = true // if this is true banchmark test works.
)

const (
	appId             = "P-JV2nVIRUtgyPO5xRNeYll2mT4F5QG4bS"
	urlConst          = "https://api.admin.u-code.io"
	multipleUpdateUrl = "https://api.admin.u-code.io/v1/object/multiple-update/"
)

// Handle a serverless request
func Handle(req []byte) string {
	if time.Now().Minute() == 00 && time.Now().Hour() == 01 {
		Send("START CRON JOB WORK")
		var (
			timeEnd, timeStart string
			notifRequests      = MultipleUpdateRequest{}
		)

		var (
			startT                          = time.Now()
			yearStart, monthStart, dayStart = startT.Date()
		)

		if monthStart > 10 {
			if dayStart > 10 {
				timeStart = fmt.Sprintf("%d-%d-%dT23:59:59.000Z", yearStart, monthStart, dayStart)
			} else {
				timeStart = fmt.Sprintf("%d-%d-0%dT23:59:59.000Z", yearStart, monthStart, dayStart)
			}
		} else {
			if dayStart > 10 {
				timeStart = fmt.Sprintf("%d-0%d-%dT23:59:59.000Z", yearStart, monthStart, dayStart)
			} else {
				timeStart = fmt.Sprintf("%d-0%d-0%dT23:59:59.000Z", yearStart, monthStart, dayStart)
			}
		}

		var (
			endT                      = time.Now().AddDate(0, 0, 1)
			yearEnd, monthEnd, dayEnd = endT.Date()
		)

		if monthEnd > 10 {
			if dayEnd > 10 {
				timeEnd = fmt.Sprintf("%d-%d-%dT23:59:59.000Z", yearEnd, monthEnd, dayEnd)
			} else {
				timeEnd = fmt.Sprintf("%d-%d-0%dT23:59:59.000Z", yearEnd, monthEnd, dayEnd)
			}
		} else {
			if dayEnd > 10 {
				timeEnd = fmt.Sprintf("%d-0%d-%dT23:59:59.000Z", yearEnd, monthEnd, dayEnd)
			} else {
				timeEnd = fmt.Sprintf("%d-0%d-0%dT23:59:59.000Z", yearEnd, monthEnd, dayEnd)
			}
		}

		medicineTakingsResp := GetListClientApiResponse{}

		// get object slim list of medicine_taking
		medicineTakingsRespByte, err := DoRequest(baseUrl+fmt.Sprintf(`/v2/object-slim/get-list/medicine_taking?data={"with_relations":true,"frequency":["always"],"last_time":{"$gte":"%s","$lt":"%s"}}`, timeStart, timeEnd), "GET", nil, appId)
		if err != nil {
			return Handler("error", err.Error())
		}

		if err = json.Unmarshal(medicineTakingsRespByte, &medicineTakingsResp); err != nil {
			return Handler("error", err.Error())
		}

		res11111111, _ := json.Marshal(medicineTakingsResp)

		Send("medicineTakingsResp!!!!!!!!!!" + string(res11111111))

		var (
			medicineTakings = medicineTakingsResp.Data.Data.Response
			requests        = []map[string]interface{}{}
			lastTimeReq     = []map[string]interface{}{}
		)

		for _, medicineTaking := range medicineTakings {
			var hours map[string]interface{}

			medicineTakingID := cast.ToString(medicineTaking["guid"])

			jsBody := cast.ToString(medicineTaking["json_body"])

			if err = json.Unmarshal([]byte(jsBody), &hours); err != nil {
				return Handler("error", err.Error())
			}

			var hoursOfDay = cast.ToSlice(hours["hours_of_day"])

			// sort hours of the day
			sort.Slice(hoursOfDay, func(i, j int) bool {
				return hoursOfDay[i].(string) < hoursOfDay[j].(string)
			})

			for i, hour := range hoursOfDay {
				var hourStr = cast.ToString(hour)

				lastTimeStr := endT.Format("2006-01-02") + "T" + hourStr + ".000Z"
				lastTime, _ := time.Parse("2006-01-02T15:04:05.000Z", lastTimeStr)

				date := lastTime.Add(19 * time.Hour).Format("2006-01-02T15:04:05.000Z")

				requests = append(requests, map[string]interface{}{
					"naznachenie_id":     cast.ToString(medicineTaking["naznachenie_id"]),
					"medicine_taking_id": medicineTakingID,
					"time_take":          date,
					"before_after_food":  cast.ToString(cast.ToSlice(medicineTaking["description"])[0]),
					"cleints_id":         cast.ToString(medicineTaking["cleints_id"]),
					"preparati_id":       cast.ToString(medicineTaking["preparati_id"]),
					"is_from_patient":    cast.ToBool(medicineTaking["is_from_patient"]),
					"count":              cast.ToInt(medicineTaking["count"]),
					"preparat_name":      cast.ToString(medicineTaking["preparat_name"]),
				})

				if i == len(hoursOfDay)-1 {
					lastTimeReq = append(lastTimeReq, map[string]interface{}{
						"guid":      medicineTakingID,
						"last_time": date,
					})
				}

				notifRequests.Data.Objects = append(notifRequests.Data.Objects, map[string]interface{}{
					"client_id":    cast.ToString(medicineTaking["cleints_id"]),
					"title":        "Время принятия препарата!",
					"body":         "Вам назначен препарат: ",
					"title_uz":     "Preparatni qabul qilish vaqti bo'ldi!",
					"body_uz":      "Sizga preparat tayinlangan: ",
					"is_read":      false,
					"preparati_id": cast.ToString(medicineTaking["preparati_id"]),
					"time_take":    date,
				})
			}
		}

		// multiple update of patient_medication
		if _, err = DoRequest(baseUrl+"/v1/object/multiple-update/patient_medication", "PUT", Request{Data: map[string]interface{}{"objects": requests}}, appId); err != nil {
			return Handler("error", err.Error())
		}

		// multiple update last_time of medicine_taking
		if _, err = DoRequest(baseUrl+"/v1/object/multiple-update/medicine_taking", "PUT", Request{Data: map[string]interface{}{"objects": lastTimeReq}}, appId); err != nil {
			return Handler("error", err.Error())
		}

		// multiple update notifications
		if _, err = DoRequest(baseUrl+"/v1/object/multiple-update/notifications", "PUT", notifRequests, appId); err != nil {
			return Handler("error", err.Error())
		}
	}

	return ""
}

func MultipleUpdateObject(url, tableSlug, appId string, request Request) error {
	_, err := DoRequest(url+"/v1/object/multiple-update/"+tableSlug, "PUT", request, appId)
	if err != nil {
		return errors.New("error while updating multiple objects" + err.Error())
	}
	return nil
}

func GetListObject(url, tableSlug, appId string, request Request) (GetListClientApiResponse, Response, error) {
	response := Response{}

	getListResponseInByte, err := DoRequest(url+"/v1/object/get-list/"+tableSlug+"?from-ofs=true", "POST", request, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while getting list of object"}
		response.Status = "error"
		return GetListClientApiResponse{}, response, errors.New("error")
	}
	var getListObject GetListClientApiResponse
	err = json.Unmarshal(getListResponseInByte, &getListObject)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while unmarshalling get list object"}
		response.Status = "error"
		return GetListClientApiResponse{}, response, errors.New("error")
	}
	return getListObject, response, nil
}

func GetSingleObject(url, tableSlug, appId, guid string) (ClientApiResponse, Response, error) {
	response := Response{}

	var getSingleObject ClientApiResponse
	getSingleResponseInByte, err := DoRequest(url+"/v1/object/{table_slug}/{guid}?from-ofs=true", "GET", nil, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while getting single object"}
		response.Status = "error"
		return ClientApiResponse{}, response, errors.New("error")
	}
	err = json.Unmarshal(getSingleResponseInByte, &getSingleObject)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while unmarshalling single object"}
		response.Status = "error"
		return ClientApiResponse{}, response, errors.New("error")
	}
	return getSingleObject, response, nil
}

func CreateObject(url, tableSlug, appId string, request Request) (Datas, Response, error) {
	response := Response{}

	var createdObject Datas
	createObjectResponseInByte, err := DoRequest(url+"/v1/object/"+tableSlug+"?from-ofs=true&project-id=a4dc1f1c-d20f-4c1a-abf5-b819076604bc", "POST", request, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while creating object"}
		response.Status = "error"
		return Datas{}, response, errors.New("error")
	}
	err = json.Unmarshal(createObjectResponseInByte, &createdObject)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while unmarshalling create object object"}
		response.Status = "error"
		return Datas{}, response, errors.New("error")
	}
	return createdObject, response, nil
}

func UpdateObject(url, tableSlug, appId string, request Request) (Response, error) {
	response := Response{}

	_, err := DoRequest(url+"/v1/object/{table_slug}?from-ofs=true", "PUT", request, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while updating object"}
		response.Status = "error"
		return response, errors.New("error")
	}
	return response, nil
}

func UpdateObjectMany2Many(url, appId string, request RequestMany2Many) (Response, error) {
	response := Response{}

	_, err := DoRequest(url+"/v1/many-to-many/", "PUT", request, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while updating object"}
		response.Status = "error"
		return response, errors.New("error")
	}
	return response, nil
}

func DeleteObject(url, tableSlug, appId, guid string) (Response, error) {
	response := Response{}

	_, err := DoRequest(url+"/v1/object/{table_slug}/{guid}?from-ofs=true", "DELETE", Request{}, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while updating object"}
		response.Status = "error"
		return response, errors.New("error")
	}
	return response, nil
}

func DoRequest(url string, method string, body interface{}, appId string) ([]byte, error) {
	data, err := json.Marshal(&body)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: time.Duration(30 * time.Second),
	}
	request, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	request.Header.Add("authorization", "API-KEY")
	request.Header.Add("X-API-KEY", appId)

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respByte, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respByte, nil
}

func Handler(status, message string) string {
	var (
		response Response
		Message  = make(map[string]interface{})
	)

	Send(status + message)
	response.Status = status
	Message["message"] = message
	response.Data = Message
	respByte, _ := json.Marshal(response)
	return string(respByte)
}

func Send(text string) {
	var (
		bot, _ = tgbotapi.NewBotAPI(botToken)
		chatID = int64(chatID)
		msg    = tgbotapi.NewMessage(chatID, fmt.Sprintf("message from madad payme route function: %s", text))
	)

	bot.Send(msg)
}

type DateObject struct {
	Day   string
	Hour  string
	Dates []string
}

// Datas This is response struct from create
type Datas struct {
	Data struct {
		Data struct {
			Data map[string]interface{} `json:"data"`
		} `json:"data"`
	} `json:"data"`
}

// ClientApiResponse This is get single api response
type ClientApiResponse struct {
	Data ClientApiData `json:"data"`
}

type ClientApiData struct {
	Data ClientApiResp `json:"data"`
}

type ClientApiResp struct {
	Response map[string]interface{} `json:"response"`
}

type Response struct {
	Status string                 `json:"status"`
	Data   map[string]interface{} `json:"data"`
}

type HttpRequest struct {
	Method  string      `json:"method"`
	Path    string      `json:"path"`
	Headers http.Header `json:"headers"`
	Params  url.Values  `json:"params"`
	Body    []byte      `json:"body"`
}

type AuthData struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

type NewRequestBody struct {
	RequestData HttpRequest            `json:"request_data"`
	Auth        AuthData               `json:"auth"`
	Data        map[string]interface{} `json:"data"`
}
type Request struct {
	Data map[string]interface{} `json:"data"`
}

type RequestMany2Many struct {
	IdFrom    string   `json:"id_from"`
	IdTo      []string `json:"id_to"`
	TableFrom string   `json:"table_from"`
	TableTo   string   `json:"table_to"`
}

// GetListClientApiResponse This is get list api response
type GetListClientApiResponse struct {
	Data GetListClientApiData `json:"data"`
}

type GetListClientApiData struct {
	Data GetListClientApiResp `json:"data"`
}

type GetListClientApiResp struct {
	Response []map[string]interface{} `json:"response"`
}

//

type CustomDataObj struct {
	CycleName  string   `json:"cycle_name"`
	CycleCount int      `json:"cycle_count"`
	Time       string   `json:"time"`
	Dates      []string `json:"dates"`
}

type DayTime struct {
	Day  string `json:"day"`
	Time string `json:"time"`
}
type DateTime struct {
	Date string `json:"date"`
	Time string `json:"time"`
}

type Medicine struct {
	Type            string        `json:"type"`
	DayData         []string      `json:"dayData"`
	CustomData      CustomDataObj `json:"customData"`
	WeekData        []DayTime     `json:"weekData"`
	MonthData       []DateTime    `json:"monthData"`
	BeforeAfterFood string        `json:"before_after_food"`
	StartDate       string        `json:"start_date"`
	EndDate         string        `json:"end_date"`
	CurrentAmount   int           `json:"current_amount"`
	DaysOfWeek      []int         `json:"days_of_week"`
	HoursOfDay      []string      `json:"hours_of_day"`
	WithoutBreak    bool          `json:"without_break"`
}

type MultipleUpdateRequest struct {
	Data struct {
		Objects []map[string]interface{} `json:"objects"`
	} `json:"data"`
}

func SortHours(timeStrings []string) ([]time.Time, error) {
	// Parse the time strings into time.Time objects
	times := make([]time.Time, len(timeStrings))
	for i, str := range timeStrings {
		parsedTime, err := time.Parse("15:04:05", str)
		if err != nil {
			return nil, err
		}
		parsedTime = parsedTime.Add(time.Hour * -5)
		times[i] = parsedTime
	}

	// Sort the time.Time objects
	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})

	return times, nil
}

func GetNextDate(current time.Time, days []int, times []time.Time) time.Time {
	nextDate := current

	// Get next hour
	var nextTime time.Time

	for _, t := range times {
		if t.Hour() == current.Hour() {
			if t.Minute() > current.Minute() {
				nextTime = t
				break
			}
		} else if t.Hour() > current.Hour() {
			nextTime = t
			break
		}
	}

	if nextTime == (time.Time{}) {
		nextTime = times[0]
		nextDate = nextDate.AddDate(0, 0, 1)
	}
	// current day of the week
	currentDay := int(nextDate.Weekday())

	// iterate days array and find next upcoming day
	addition := -1
	for _, day := range days {
		if day >= currentDay {
			addition = day - currentDay
			nextDate = nextDate.AddDate(0, 0, day-currentDay)
			break
		}
	}
	if addition == -1 {
		nextDate = nextDate.AddDate(0, 0, days[0]+7-currentDay)
	}

	// Combine the next date and time
	nextDateTime := time.Date(nextDate.Year(), nextDate.Month(), nextDate.Day(), nextTime.Hour(), nextTime.Minute(), nextTime.Second(), 0, nextDate.Location())
	return nextDateTime
}
