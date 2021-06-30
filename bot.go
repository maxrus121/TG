package main

/*
Заготовка кода для бота
*/
import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	refreshRate           = 3
	baseTelegramUrl       = "https://api.telegram.org"
	getUpdatesUrl         = "getUpdates"
	sendMessageUrl        = "sendMessage"
	telegramToken         = "1565772755:AAFn-yqTceIZOfi5feF5kD5KNfnQxkCxNoI"
	defaultHandlerMessage = "_default"
)

/*
Мне нужно чтобы мапа передовалась в обработчик из диспетчера
и чтобы меняя информацию в обработчике она менялась и во вне
Я передал просто *map[int]Room, но как оказалось ссылка на мапу
передалась, а вот значения внутри нет(точнее их копия)
*/

type MainMessageHandler func(UpdateResultMessageT)

//Описываем структуры Telegram

type UpdateT struct {
	Ok     bool            `json:"ok"`
	Result []UpdateResultT `json:"result"`
}

type UpdateResultT struct {
	UpdateId int                  `json:"update_id"`
	Message  UpdateResultMessageT `json:"message"`
}
type UpdateResultMessageT struct {
	MessageId int               `json:"message_id"`
	From      UpdateResultFromT `json:"from"`
	Chat      UpdateResultChatT `json:"chat"`
	Date      int               `json:"date"`
	Text      string            `json:"text"`
}

type UpdateResultFromT struct {
	Id        int    `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Language  string `json:"language_code"`
}

type UpdateResultChatT struct {
	Id        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Type      string `json:"type"`
}

type SendMessageResponseT struct {
	Ok     bool               `json:"ok"`
	Result ResultSendMessageT `json:"result"`
}

type ResultSendMessageT struct {
	MessageID int                `json:"message_id"`
	From      FromResultMessageT `json:"from"`
}

type FromResultMessageT struct {
	Id        int    `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type MessageSend struct {
	ChatID   int                 `json:"chat_id"`
	Text     string              `json:"text"`
	Keyboard ReplyKeyboardMarkup `json:"reply_markup"`
}
type KeyboardButton struct {
	Text            string `json:"text"`
	RequestContact  bool   `json:"request_contact"`
	RequestLocation bool   `json:"request_location"`
}

type ReplyKeyboardMarkup struct {
	Keyboard        [][]KeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool               `json:"resize_keyboard"`   // optional
	OneTimeKeyboard bool               `json:"one_time_keyboard"` // optional
	Selective       bool               `json:"selective"`         // optional
}

//ФУНКЦИИ ДЛЯ ВЗАИМОДЕЙСТВИЯ С ТЕЛЕГРАММ
func getUpdates(offset int) (UpdateT, error) {
	//Метод получения обновлений с API с какого сообщения начать
	method := getUpdatesUrl
	if offset != 0 {
		method += "?offset=" + strconv.Itoa(offset)
	}
	response := sendRequest(method, []byte{0})
	update := UpdateT{}
	err := json.Unmarshal(response, &update)

	if err != nil {
		return update, err
	}
	return update, nil
}

func sendMessage(chatId int, text string, key ReplyKeyboardMarkup) (SendMessageResponseT, error) {
	messageStruct := &MessageSend{
		ChatID:   chatId,
		Text:     text,
		Keyboard: key,
	}
	jsonMessage, err := json.Marshal(messageStruct)
	if err != nil {
		log.Println(err.Error())
	}
	//Метод отправки сообщений пользователю
	method := sendMessageUrl
	response := sendRequest(method, jsonMessage)
	sendMessage := SendMessageResponseT{}

	err = json.Unmarshal(response, &sendMessage)

	if err != nil {
		return sendMessage, err
	}
	return sendMessage, nil
}

func sendRequest(method string, jsonMessage []byte) []byte {
	//Метод отправки http запроса
	sendURL := baseTelegramUrl + "/bot" + telegramToken + "/" + method
	response := make([]byte, 0)
	resp, err := http.Post(sendURL, "application/json", bytes.NewBuffer(jsonMessage))
	if err != nil {
		log.Println(err)
		return response
	}
	defer resp.Body.Close()
	for true {
		bs := make([]byte, 1024)
		n, err := resp.Body.Read(bs)
		response = append(response, bs[:n]...)

		if n == 0 || err != nil {
			break
		}
	}
	return response
}

//ФУНКЦИИ ДЛЯ СОЕДИНЕНИЯ С ГУГЛ ТАБЛИЦАМИ
// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func write(Information []interface{}, writeRange string) {
	//ctx := context.Background()
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/sheets.googleapis.com-go-quickstart.json
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets Client %v", err)
	}

	spreadsheetId := "1uz21FOMfX7GxhKRJSj-E1UBnwd-IF6qjbJgDuiHUrwI"

	var vr sheets.ValueRange

	vr.Values = append(vr.Values, Information)

	_, err = srv.Spreadsheets.Values.Update(spreadsheetId, writeRange, &vr).ValueInputOption("RAW").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet. %v", err)
	}

}

func read(readRange string) (result []interface{}) {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	// https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit
	//1uz21FOMfX7GxhKRJSj-E1UBnwd-IF6qjbJgDuiHUrwI
	spreadsheetId := "1uz21FOMfX7GxhKRJSj-E1UBnwd-IF6qjbJgDuiHUrwI"
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
	} else {
		for _, row := range resp.Values {
			// Print columns A and E, which correspond to indices 0 and 4.
			result = append(result, row)
		}
	}
	return result
}

//ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ

func ToGenericArray(arr ...interface{}) []interface{} {
	return arr
}

func init() {
	//Инициируем наш логгер
	file, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(file)
	log.SetPrefix("TeleBOT: ")
	log.SetFlags(log.Ldate | log.Ltime)
	log.Println("Start work bot")
}

func remove(s []interface{}, i int) []interface{} {
	s[i] = s[len(s)-1]
	// We do not need to put s[i] at the end, as it will be discarded anyway
	return s[:len(s)-1]
}

func intInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func NewReplyKeyboard(rows ...[]KeyboardButton) ReplyKeyboardMarkup {
	var keyboard [][]KeyboardButton

	keyboard = append(keyboard, rows...)

	return ReplyKeyboardMarkup{
		ResizeKeyboard: true,
		Keyboard:       keyboard,
	}
}

func NewKeyboardButtonRow(buttons ...KeyboardButton) []KeyboardButton {
	var row []KeyboardButton

	row = append(row, buttons...)

	return row
}

//ФУНКЦИИ ДЛЯ ОБРАБОТКИ КОМАНД В БОТЕ
func startHandler(message UpdateResultMessageT) {
	// Обработчик команды /start
	userMessage := "Привет " + message.From.FirstName + "!\n" + "Набери сообщение /help для просмотра команд"
	_, err := sendMessage(message.Chat.Id, userMessage, KeyBoard0())
	if err != nil {
		log.Println(err.Error())
	}
}

func helpHandler(message UpdateResultMessageT) {
	helpMessage := "Вот что я умею \n" +
		"/help - Показать меню помощи\n" +
		"/want_a_meeting - Режим ожидания встречи\n" +
		"/status - Показать информацию по моему статусу\n" +
		"/quit - Выйти из режима ожидания встречи\n"
	_, err := sendMessage(message.Chat.Id, helpMessage, KeyBoard0())
	if err != nil {
		log.Println(err.Error())
	}
}

func meetHandler(message UpdateResultMessageT) {
	if message.From.Username != "" {
		res := read("БД!A1:E10")
		flag := 0
		for i := 1; i < len(res); i++ {
			if res[i].([]interface{})[2] == message.From.Username {
				log.Println("Found in database")
				flag = i
			}
		}
		if flag != 0 {
			write(ToGenericArray("Ожидаю встречу"), "БД!F"+strconv.Itoa(flag+1))
			_, err := sendMessage(message.Chat.Id, "Данные обновлены", KeyBoard0())
			if err != nil {
				log.Println(err.Error())
			}
		} else {
			write(ToGenericArray(message.From.FirstName, message.From.LastName, message.From.Username, message.From.Id, "Ожидаю встречу"), "БД!A"+strconv.Itoa(len(res)+1))
			_, err := sendMessage(message.Chat.Id, "Данные добавлены", KeyBoard0())
			if err != nil {
				log.Println(err.Error())
			}
		}
	} else {
		_, err := sendMessage(message.Chat.Id, "У твоего аккаунта Telegram не задано имя пользователя без него "+
			"другие пользователи не смогут с тобой связаться. Пожалуйста открой настройки и нажми на свою "+
			"фотографию, после этого откроется окно в котором можно задать своё имя пользователя. Как сделаешь "+
			"возвращайся и мы попробуем снова!", KeyBoard0())
		if err != nil {
			log.Println(err.Error())
		}
	}
}

func quitHandler(message UpdateResultMessageT) {
	res := read("БД!A1:E10")
	flag := 0
	for i := 1; i < len(res); i++ {
		if res[i].([]interface{})[2] == message.From.Username {
			log.Println("Found in database")
			flag = i
		}
	}
	if flag != 0 {
		write(ToGenericArray("Не хочу встречаться"), "БД!E"+strconv.Itoa(flag+1))
		_, err := sendMessage(message.Chat.Id, "Данные обновлены", KeyBoard0())
		if err != nil {
			log.Println(err.Error())
		}
	} else {
		_, err := sendMessage(message.Chat.Id, "Не нашел тебя в базе", KeyBoard0())
		if err != nil {
			log.Println(err.Error())
		}
	}
}

func statusHandler(message UpdateResultMessageT) {
	res := read("БД!A1:E10")
	flag := 0
	for i := 1; i < len(res); i++ {
		if res[i].([]interface{})[2] == message.From.Username {
			log.Println("Found in database")
			flag = i
		}
	}
	if flag != 0 {
		_, err := sendMessage(message.Chat.Id, "Ваш статус: "+res[flag].([]interface{})[4].(string), KeyBoard0())
		if err != nil {
			log.Println(err.Error())
		}
	} else {
		_, err := sendMessage(message.Chat.Id, "Не нашел тебя в базе", KeyBoard0())
		if err != nil {
			log.Println(err.Error())
		}
	}
}

func defaultMainHandler(message UpdateResultMessageT) {
	result := read("Встречи!A1:G10")
	flag := 0
	for i := 1; i < len(result); i++ {
		if message.From.Username == result[i].([]interface{})[0] {
			if result[i].([]interface{})[5].(string) == "Ожидается" {
				write(ToGenericArray(message.Text), "Встречи!F"+strconv.Itoa(i+1))
				flag = 1
			} else {
				write(ToGenericArray(result[i].([]interface{})[5].(string)+message.Text), "Встречи!F"+strconv.Itoa(i+1))
				flag = 1
			}
			_, err := sendMessage(message.Chat.Id, "Я записал это как Feedback", KeyBoard0())
			if err != nil {
				log.Println(err.Error())
			}
		} else if message.From.Username == result[i].([]interface{})[2] {
			if result[i].([]interface{})[6].(string) == "Ожидается" {
				write(ToGenericArray(message.Text), "Встречи!G"+strconv.Itoa(i+1))
				flag = 1
			} else {
				write(ToGenericArray(result[i].([]interface{})[6].(string)+message.Text), "Встречи!G"+strconv.Itoa(i+1))
				flag = 1
			}
			_, err := sendMessage(message.Chat.Id, "Я записал это как Feedback", KeyBoard0())
			if err != nil {
				log.Println(err.Error())
			}
		}
	}
	randomMessages := []string{
		"Я даже и не знаю, что сказать(",
		"Возможно я не понял твою команду",
	}
	if flag == 0 {
		_, err := sendMessage(message.Chat.Id, randomMessages[rand.Intn(len(randomMessages))], KeyBoard0())
		if err != nil {
			log.Println(err.Error())
		}
	}
}

//ВНУТРЕННИЕ ФУНКЦИИ БОТА НЕ ВЫЗЫВАЕМЫЕ НАПРЯМУЮ
func KeyBoard0() ReplyKeyboardMarkup {
	button := &KeyboardButton{
		Text:            "🧐 Помощь",
		RequestContact:  false,
		RequestLocation: false,
	}
	button1 := &KeyboardButton{
		Text:            "👫 Хочу встречу",
		RequestContact:  false,
		RequestLocation: false,
	}
	button2 := &KeyboardButton{
		Text:            "👀 Статус",
		RequestContact:  false,
		RequestLocation: false,
	}
	button3 := &KeyboardButton{
		Text:            "🙅 Выход",
		RequestContact:  false,
		RequestLocation: false,
	}
	key := NewKeyboardButtonRow(*button, *button1)
	key1 := NewKeyboardButtonRow(*button2, *button3)
	keyboard := NewReplyKeyboard(key, key1)
	return keyboard
}

func callAt(hour, min, sec int) error {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return err
	}

	// Вычисляем время первого запуска.
	now := time.Now().Local()
	firstCallTime := time.Date(
		now.Year(), now.Month(), now.Day(), hour, min, sec, 0, loc)
	if firstCallTime.Before(now) {
		// Если получилось время раньше текущего, прибавляем сутки.
		firstCallTime = firstCallTime.Add(time.Hour * 24)
	}

	// Вычисляем временной промежуток до запуска.
	duration := firstCallTime.Sub(time.Now().Local())
	go func() {
		time.Sleep(duration)
		for {
			if time.Now().Weekday() == time.Saturday {
				generateMeeting()
			} else if time.Now().Weekday() == time.Thursday {
				giveFeedback()
			}
			// Следующий запуск через сутки.
			time.Sleep(time.Hour * 24)
		}
	}()

	return nil
}

func giveFeedback() {
	res := read("Встречи!A1:G10")
	for i := 1; i < len(res); i++ {
		chatId, _ := strconv.Atoi(res[i].([]interface{})[1].(string))
		_, err := sendMessage(chatId, "Привет! Расскажи мне как прошла встреча. Эта информация поможет всем стать лучше!", KeyBoard0())
		if err != nil {
			log.Println(err.Error())
		}
		chatId, _ = strconv.Atoi(res[i].([]interface{})[3].(string))
		_, err = sendMessage(chatId, "Привет! Расскажи мне как прошла встреча. Эта информация поможет всем стать лучше!", KeyBoard0())
		if err != nil {
			log.Println(err.Error())
		}
	}
}

func meetingToArchive() {
	res := read("Встречи!A1:G10")
	ram := read("Архив встреч!A1:D20")
	for i := 1; i < len(res); i++ {
		if len(res[i].([]interface{})) == 7 {
			write(ToGenericArray(res[i].([]interface{})[0], res[i].([]interface{})[2], res[i].([]interface{})[4], res[i].([]interface{})[5], res[i].([]interface{})[6]), "Архив встреч!A"+strconv.Itoa(len(ram)+i))
			write(ToGenericArray("", "", "", "", "", "", ""), "Встречи!A"+strconv.Itoa(i+1))
		} else {
			res[i] = append(res[i].([]interface{}), "Нет информации", "Нет информации", "Нет информации")
			write(ToGenericArray(res[i].([]interface{})[0], res[i].([]interface{})[2], res[i].([]interface{})[4], res[i].([]interface{})[5], res[i].([]interface{})[6]), "Архив встреч!A"+strconv.Itoa(len(ram)+i))
			write(ToGenericArray("", "", "", "", "", "", ""), "Встречи!A"+strconv.Itoa(i+1))
		}
	}
}

func generateMeeting() {
	meetingToArchive()
	res := read("БД!A1:E10")
	for i := 1; i < len(res); i++ {
		if res[i].([]interface{})[4] == "Не хочу встречаться" {
			remove(res, i)
			log.Println("Delete person")
		}
	}
	ram := read("Встречи!A1:D10")
	person := randomCreate(len(res) - 1)
	for i := 0; i < len(person); i++ {
		if i%2 == 0 { //Проверить правильность расчета строки
			write(ToGenericArray(res[person[i]].([]interface{})[2], res[person[i]].([]interface{})[3]), "Встречи!A"+strconv.Itoa(len(ram)+1+i/2))
			date1 := time.Now().Month()
			date2 := time.Now().Day()
			write(ToGenericArray(strconv.Itoa(date2)+"."+strconv.Itoa(int(date1)), "Ожидается", "Ожидается"), "Встречи!E"+strconv.Itoa(len(ram)+1+i/2))
		} else {
			write(ToGenericArray(res[person[i]].([]interface{})[2], res[person[i]].([]interface{})[3]), "Встречи!C"+strconv.Itoa(len(ram)+1+i/2))
		}
	}
	ram = read("Встречи!A1:D10")
	for i := 1; i < len(ram); i++ {
		chatId, _ := strconv.Atoi(ram[i].([]interface{})[1].(string))
		_, err := sendMessage(chatId, "Привет! На этой неделе твой партнер @"+ram[i].([]interface{})[2].(string), KeyBoard0())
		if err != nil {
			log.Println(err.Error())
		}
		chatId, _ = strconv.Atoi(ram[i].([]interface{})[3].(string))
		_, err = sendMessage(chatId, "Привет! на этой неделе твой партнер @"+ram[i].([]interface{})[0].(string), KeyBoard0())
		if err != nil {
			log.Println(err.Error())
		}
	}
}

func randomCreate(numberOfPersons int) []int {
	var person []int
	person = append(person, rand.Intn(numberOfPersons)+1)
	for i := 0; len(person) < numberOfPersons; i++ {
		rnd1 := rand.Intn(numberOfPersons) + 1
		if intInSlice(rnd1, person) == false {
			person = append(person, rnd1)
		}
	}
	return person
}

func main() {
	//Идея вынимать из мапы функции и по единому интерфейсу взаимодествовать
	// с обработчиками
	mainDispatcher := map[string]MainMessageHandler{
		"/start":              startHandler,
		"/help":               helpHandler,
		"/want_a_meeting":     meetHandler,
		"/quit":               quitHandler,
		"/status":             statusHandler,
		"🧐":                   helpHandler,
		"👀":                   statusHandler,
		"👫":                   meetHandler,
		"🙅":                   quitHandler,
		defaultHandlerMessage: defaultMainHandler,
	}

	//Необходим для получения обновлений.
	offset := 0
	//Вызов функции создания встреч
	err := callAt(21, 56, 0)
	if err != nil {
		log.Println("error in calling function" + err.Error())
	}

	// Эмуляция дальнейшей работы программы.

	for {
		//Спим определнное количество секунд
		time.Sleep(1000000000 * refreshRate)

		//Получаем список обновлений
		update, err := getUpdates(offset)
		if err != nil {
			log.Println("error while receiving updates: " + err.Error())
			continue
		}

		// Нет обновлений? пропускаем
		updatesLen := len(update.Result)
		if updatesLen == 0 {
			continue
		}

		log.Println("received " + strconv.Itoa(updatesLen) + " messages")

		//Проходим циклом по обновлениям и распределяем по диспетчерам
		for _, item := range update.Result {
			//Если прислать стикер, то текст будет пустым
			if item.Message.Text == "" {
				continue
			}
			// Обрабатываем только первое слово. Тоесть команду
			command := strings.Fields(item.Message.Text)[0]

			//Проверяем комнату пользователя. Если она есть, то пытаемся получить обработчик из диспетчера комнат
			if value, KeyExists := mainDispatcher[command]; KeyExists {
				value(item.Message)
			} else {
				mainDispatcher[defaultHandlerMessage](item.Message)
			}
		}
		//Выставляем offset значение +1 от максимального ID события
		offset = update.Result[updatesLen-1].UpdateId + 1
	}
}
