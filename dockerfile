# Используем базовый образ для Go
FROM golang:latest

# Создадим директорию
RUN mkdir /app

# Скопируем всё в директорию
ADD . /app/

# Установим рабочей папкой директорию
WORKDIR /app

RUN go mod init github.com/maxrus121/TG
# Получим зависимости, которые использовали в боте
Run go get golang.org/x/net/context
Run go get golang.org/x/oauth2
Run go get golang.org/x/oauth2/google
Run go get google.golang.org/api/sheets/v4
# Соберём приложение
RUN go build -o main .

# Запустим приложение
CMD ["/app/main"]