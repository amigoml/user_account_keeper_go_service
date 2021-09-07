
[<!--lint ignore no-dead-urls-->![GitHub Actions status | sdras/awesome-actions](https://github.com/amigoml/account_keeper/workflows/CI/badge.svg)](https://github.com/amigoml/account_keeper/actions?workflow=CI)


## Задача

[отсюда](https://github.com/avito-tech/autumn-2021-intern-assignment)

1. Сервис биллинга с помощью внешних мерчантов (аля через visa/mastercard) обработал зачисление денег на наш счет. 
Теперь биллингу нужно добавить эти деньги на баланс пользователя.
2. Пользователь хочет купить у нас какую-то услугу. Для этого у нас есть специальный сервис управления услугами, 
который перед применением услуги проверяет баланс и потом списывает необходимую сумму.
3. В ближайшем будущем планируется дать пользователям возможность перечислять деньги друг-другу внутри нашей платформы. 
Мы решили заранее предусмотреть такую возможность и заложить ее в архитектуру нашего сервиса.

Реализовано на Go и Postgresql


---------------------------------------------------
## Ручки

- get_balance(user_id)
Метод получения текущего баланса пользователя. Принимает id пользователя. Баланс всегда в рублях.

- top_up_balance(user_id, accrued_amount)
Метод начисления средств на баланс. Принимает id пользователя и сколько средств зачислить.

- write_off_money(user_id, debited_amount)
Метод списания средств с баланса. Принимает id пользователя и сколько средств списать.

- transfer_money(from_user_id, to_user_id, amount)
Метод перевода средств от пользователя к пользователю. 
Принимает id пользователя с которого нужно списать средства, id пользователя которому должны зачислить средства, а также сумму.

- get_user_history(user_id, n_last_operations)
Возвращает последние операции пользователя.

---------------------------------------------------
## Запуск

`source run.sh`

## Примеры запросов
```
curl -X GET "http://127.0.0.1:3000/get_balance?user_id=3"
curl -X GET "http://127.0.0.1:3000/get_balance?user_id=4"
curl -X GET "http://127.0.0.1:3000/top_up_balance?user_id=3&accrued_amount=100"
curl -X GET "http://127.0.0.1:3000/write_off_money?user_id=3&debited_amount=1"
curl -X GET "http://127.0.0.1:3000/transfer_money?from_user_id=3&to_user_id=4&amount=66"
curl -X GET "http://127.0.0.1:3000/get_balance?user_id=4"
curl -X GET "http://127.0.0.1:3000/get_balance?user_id=3"
curl -X GET "http://127.0.0.1:3000/get_user_history?user_id=3&n_last_operations=5"
```
---------------------------------------------------

## Структура БД

tables

    account:
        - user_id : int
        - current_sum : decimal
    
    history:
        - id : int
        - user_id : int
        - is_debit : bool
        - amount : decimal
        - time : DateTime

--------------
## Что не сделано

- нет адекватной обработка ошибок, человекочитаемого описания в ответе и хттп коды
- не написаны нормально тесты
- нет работы с валютой
- нет нормальной истории операций с пагинацией и сортировками



## Полезные ссылки

https://github.com/jdaarevalo/docker_postgres_with_data postgres docker

https://golangdocs.com/golang-postgresql-example

https://gist.github.com/divan/eb11ddc97aab765fb9b093864410fd25

https://tutorialedge.net/golang/creating-simple-web-server-with-golang/#serving-content-over-https

https://golang.org/doc/database/execute-transactions go transactions 

https://pkg.go.dev/database/sql#DB.QueryRowContext go working with db

https://golangdocs.com/golang-postgresql-example go db example 

