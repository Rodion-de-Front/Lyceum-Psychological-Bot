package main

//подключение требуемых пакетов
import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"context"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

// структура для приходящих сообщений и обычных кнопок
type ResponseT struct {
	Ok     bool       `json:"ok"`
	Result []MessageT `json:"result"`
}

type MessageT struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID           int    `json:"id"`
			IsBot        bool   `json:"is_bot"`
			FirstName    string `json:"first_name"`
			LastName     string `json:"last_name"`
			Username     string `json:"username"`
			LanguageCode string `json:"language_code"`
		} `json:"from"`
		Chat struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Username  string `json:"username"`
			Type      string `json:"type"`
		} `json:"chat"`
		Date     int `json:"date"`
		Document struct {
			FileName  string `json:"file_name"`
			MimeType  string `json:"mime_type"`
			Thumbnail struct {
				FileID       string `json:"file_id"`
				FileUniqueID string `json:"file_unique_id"`
				FileSize     int    `json:"file_size"`
				Width        int    `json:"width"`
				Height       int    `json:"height"`
			} `json:"thumbnail"`
			Thumb struct {
				FileID       string `json:"file_id"`
				FileUniqueID string `json:"file_unique_id"`
				FileSize     int    `json:"file_size"`
				Width        int    `json:"width"`
				Height       int    `json:"height"`
			} `json:"thumb"`
			FileID       string `json:"file_id"`
			FileUniqueID string `json:"file_unique_id"`
			FileSize     int    `json:"file_size"`
		} `json:"document"`
		Contact struct {
			PhoneNumber string `json:"phone_number"`
		} `json:"contact"`
		Location struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"location"`
		Text string `json:"text"`
		Data string `json:"data"`
	} `json:"message"`
}

// структура для инлайн кнопок
type ResponseInlineT struct {
	Ok     bool             `json:"ok"`
	Result []MessageInlineT `json:"result"`
}

type MessageInlineT struct {
	UpdateID      int `json:"update_id"`
	CallbackQuery struct {
		ID   string `json:"id"`
		From struct {
			ID           int    `json:"id"`
			IsBot        bool   `json:"is_bot"`
			FirstName    string `json:"first_name"`
			Username     string `json:"username"`
			LanguageCode string `json:"language_code"`
		} `json:"from"`
		Message struct {
			MessageID int `json:"message_id"`
			From      struct {
				ID        int64  `json:"id"`
				IsBot     bool   `json:"is_bot"`
				FirstName string `json:"first_name"`
				Username  string `json:"username"`
			} `json:"from"`
			Chat struct {
				ID        int    `json:"id"`
				FirstName string `json:"first_name"`
				Username  string `json:"username"`
				Type      string `json:"type"`
			} `json:"chat"`
			Date        int    `json:"date"`
			Text        string `json:"text"`
			ReplyMarkup struct {
				InlineKeyboard [][]struct {
					Text         string `json:"text"`
					CallbackData string `json:"callback_data"`
				} `json:"inline_keyboard"`
			} `json:"reply_markup"`
		} `json:"message"`
		ChatInstance string `json:"chat_instance"`
		Data         string `json:"data"`
	} `json:"callback_query"`
}

// структура пользователя
type UserT struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Step      int    `json:"step"`
	Tg_id     int    `json:"tg_id"`
	Entry     Entry  `json:"entry"`
}

type Entry struct {
	Name     string `json:"name"`
	RegDate  string `json:"reg_date"`
	Class    string `json:"class"`
	Time     string `json:"time"`
	Comment  string `json:"comment"`
	Username string `json:"username"`
}

// переменные для подключения к боту
var host string = "https://api.telegram.org/bot"
var token string = os.Getenv("BOT_TOKEN")

var message_link string = "https://t.me/g00ds_lol/"

// переменная для тг канала
var channelName string = os.Getenv("CHANEL_NAME")

// Идентификатор таблицы, в которую будут записываться данные
var spreadsheetID string = os.Getenv("SPREAD_SHEET_ID")

// Замените "Sheet1" на название листа, в который вы хотите добавить данные.
var sheetName string = os.Getenv("SHEET_NAME")

// данные всеx пользователей
var usersDB map[int]UserT

// главная функция работы бота
func main() {

	//достаем юзеров из кэша
	getUsers()

	//обнуление последнего id сообщения
	lastMessage := 0

	//цикл для проверки на наличие новых сообщений
	for range time.Tick(time.Second * 1) {

		//отправляем запрос к Telegram API на получение сообщений
		var url string = host + token + "/getUpdates?offset=" + strconv.Itoa(lastMessage)
		response, err := http.Get(url)
		if err != nil {
			fmt.Println(err)
		}
		data, _ := ioutil.ReadAll(response.Body)

		//посмотреть данные
		fmt.Println(string(data))

		//парсим данные из json
		var responseObj ResponseT
		json.Unmarshal(data, &responseObj)

		//парсим данные из json  (для нажатия на инлайн кнопку)
		var inline ResponseInlineT
		json.Unmarshal(data, &inline)

		//считаем количество новых сообщений
		number := len(responseObj.Result)

		//если сообщений нет - то дальше код не выполняем
		if number < 1 {
			continue
		}

		//в цикле доставать инормацию по каждому сообщению
		for i := 0; i < number; i++ {

			//обработка одного сообщения
			go processMessage(responseObj.Result[i], inline.Result[i])
		}

		//запоминаем update_id  последнего сообщения
		lastMessage = responseObj.Result[number-1].UpdateID + 1

	}
}

func getUsers() {
	//считываем из бд при включении
	dataFile, _ := ioutil.ReadFile("db.json")
	json.Unmarshal(dataFile, &usersDB)
}

// функция для отправки сообщения пользователю
func sendMessage(chatId int, text string, keyboard map[string]interface{}) {
	url := host + token + "/sendMessage?chat_id=" + strconv.Itoa(chatId) + "&text=" + text
	if keyboard != nil {
		// Преобразуем клавиатуру в JSON
		keyboardJSON, _ := json.Marshal(keyboard)
		url += "&reply_markup=" + string(keyboardJSON)
	}
	http.Get(url)
}

// // функция для отправки сообщения в канал
// func sendMessageToChanel(apiURL string) []byte {
// 	fmt.Println("sendMessage")
// 	requestURL, err := url.Parse(apiURL)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// Создание HTTP GET-запроса с параметрами
// 	request, err := http.NewRequest("GET", requestURL.String(), nil)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// Отправка запроса
// 	client := &http.Client{}
// 	response, err := client.Do(request)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer response.Body.Close()

// 	// Чтение ответа
// 	responseData, err := ioutil.ReadAll(response.Body)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// Вывод конечной ссылки запроса
// 	finalURL := request.URL.String()
// 	fmt.Println("Final URL:", finalURL)

// 	// Вывод ответа от сервера
// 	fmt.Println("Response:", string(responseData))

//		return responseData
//	}
func processMessage(message MessageT, messageInline MessageInlineT) {

	text := message.Message.Text

	chatId := 0
	if messageInline.CallbackQuery.From.ID == 0 {
		chatId = message.Message.From.ID
	} else {
		chatId = messageInline.CallbackQuery.From.ID
	}

	button := messageInline.CallbackQuery.Data

	firstName := message.Message.From.FirstName
	lastName := message.Message.From.LastName
	username := message.Message.From.Username

	//есть ли юзер
	_, exist := usersDB[chatId]
	if !exist {
		user := UserT{}
		user.ID = chatId
		user.FirstName = firstName
		user.LastName = lastName
		user.Tg_id = chatId
		user.Step = 1

		usersDB[chatId] = user

	}

	file, _ := os.Create("db.json")
	jsonString, _ := json.Marshal(usersDB)
	file.Write(jsonString)

	switch {
	// кейс для начального сообщения для пользователя
	case text == "/start" || usersDB[chatId].Step == 1 || text == "Новая запись":

		user := usersDB[chatId]
		user.Step = 1
		usersDB[chatId] = user

		// Отправляем сообщение с клавиатурой и перезаписываем шаг
		sendMessage(chatId, "Здраствуйте! Добро пожаловать в лицейского бота для записи на консультации у психолога. Введите ваше имя и фамилию", nil)

		user.Entry.Username = username
		user.Step += 1
		usersDB[chatId] = user
		break

	//кейс для выбора маркетплейса
	case usersDB[chatId].Step == 2:

		user := usersDB[chatId]

		user.Entry.Name = text

		sendMessage(chatId, "Отправьте свой класс", nil)

		user.Step += 1
		usersDB[chatId] = user

		break

	case usersDB[chatId].Step == 3:

		user := usersDB[chatId]

		user.Entry.Class = text

		sendMessage(chatId, "Отправьте удобное дату и время записи (в формате: 2023-08-28 23:00). Рабочие часы с 10-15 и с понедельника по пятницу", nil)

		user.Step += 1
		usersDB[chatId] = user
		break

	case usersDB[chatId].Step == 4:

		user := usersDB[chatId]
		user.Entry.Time = text

		//собираем объект клавиатуры для выбора языка
		buttons := [][]map[string]interface{}{
			{{"text": "Нет", "callback_data": "finish"}},
		}

		inlineKeyboard := map[string]interface{}{
			"inline_keyboard": buttons,
		}

		sendMessage(chatId, "Особые коментарии к записи", inlineKeyboard)
		user.Step += 1
		usersDB[chatId] = user
		break

	case usersDB[chatId].Step == 5 || button == "finish":

		user := usersDB[chatId]
		user.Entry.Comment = text
		// Получите текущее время
		currentTime := time.Now()

		// Определите, сколько часов вы хотите прибавить
		hoursToAdd := 3

		// Прибавьте указанное количество часов
		newTime := currentTime.Add(time.Duration(hoursToAdd) * time.Hour)

		// Определите желаемый формат времени
		format := "2006-01-02 15:04" // Например, "год-месяц-день час:минута"

		// Преобразуйте новое время в строку с помощью метода Format
		formattedTime := newTime.Format(format)

		user.Entry.RegDate = formattedTime

		// Создаем объект клавиатуры
		keyboard := map[string]interface{}{
			"keyboard": [][]map[string]interface{}{
				{
					{
						"text": "Новая запись",
					},
				},
			},
			"resize_keyboard":   true,
			"one_time_keyboard": true,
		}
		sendMessage(chatId, "Спасибо за доверие. Ирина Сергеевна подтвердит время записи", keyboard)

		usersDB[chatId] = user

		file, _ := os.Create("db.json")
		jsonString, _ := json.Marshal(usersDB)
		file.Write(jsonString)

		// Загрузите файл учетных данных вашего проекта Google Cloud в переменную `credentials`.
		credentials, err := ioutil.ReadFile("credentials.json")
		if err != nil {
			log.Fatalf("Unable to read client secret file: %v", err)
		}

		// Извлеките конфигурацию OAuth2 из файла учетных данных и получите токен доступа.
		config, err := google.JWTConfigFromJSON(credentials, sheets.SpreadsheetsScope)
		if err != nil {
			log.Fatalf("Unable to parse client secret file to config: %v", err)
		}
		client := config.Client(context.Background())

		// Создайте клиент Google Sheets API.
		sheetsService, err := sheets.New(client)
		if err != nil {
			log.Fatalf("Unable to retrieve Sheets client: %v", err)
		}

		// Загрузите файл с данными в формате JSON.
		content, err := ioutil.ReadFile("db.json")
		if err != nil {
			log.Fatalf("Unable to read data file: %v", err)
		}

		// Распарсите JSON и извлеките данные.
		var jsonData map[string]interface{}
		if err := json.Unmarshal(content, &jsonData); err != nil {
			log.Fatalf("Unable to parse JSON data: %v", err)
		}

		fmt.Println(jsonData)

		// Создайте список значений для новой строки.
		var values [][]interface{}

		// Проверяем существование и не nil поля "entry" в jsonData
		if entryData, ok := jsonData[strconv.Itoa(chatId)].(map[string]interface{}); ok && entryData != nil {

			row := []interface{}{
				entryData["entry"].(map[string]interface{})["name"],
				entryData["entry"].(map[string]interface{})["class"],
				entryData["entry"].(map[string]interface{})["time"],
				entryData["entry"].(map[string]interface{})["username"],
				entryData["entry"].(map[string]interface{})["comment"],
				entryData["entry"].(map[string]interface{})["reg_date"],
			}

			values = append(values, row)

			fmt.Println(values)

			// Создайте объект ValueRange для добавления новых строк.
			rangeValue := fmt.Sprintf("%s!A2:R", sheetName)
			vr := sheets.ValueRange{Values: values, MajorDimension: "ROWS"}
			_, err = sheetsService.Spreadsheets.Values.Append(spreadsheetID, rangeValue, &vr).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Do()
			if err != nil {
				log.Fatalf("Unable to append values: %v", err)
			}
			fmt.Println("Values appended successfully.")
		} else {
			fmt.Println("Order data not found or is nil.")
		}

		user.Entry = Entry{}
		usersDB[chatId] = user

		file, _ = os.Create("db.json")
		jsonString, _ = json.Marshal(usersDB)
		file.Write(jsonString)

		break

	}

}
