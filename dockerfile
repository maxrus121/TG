# Используем базовый образ для Go
FROM golang:latest

# Создадим директорию
RUN mkdir /app

# Скопируем всё в директорию
ADD . /app/

# Установим рабочей папкой директорию
WORKDIR /app

# Получим зависимости, которые использовали в боте
RUN go get github.com/maxrus121/TG

# Соберём приложение
RUN go build -o main .

# Запустим приложение
CMD ["/app/main"]