Эмулятор ndtp сервера для демонстрации tcpmirror (redmine_3452) в связке с ndtpClient и egtsServ

example run: 
go run main.go -p "9001" -m 0 -n 100

-p - порт, на котором слушаются соединения
-m - режим (1 - мастер)
-n - количество пакетов для получения