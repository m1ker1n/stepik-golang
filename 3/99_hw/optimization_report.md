# До оптимизации:

![img.png](img.png)

# Профилирование
```shell
# Создание бинарника теста hw3.test.exe, 
# снятие профиля для CPU cpu.out,
# профиля для памяти mem.out с флагом -memprofilerate=1 (учитывать каждую аллокацию)
go test -bench='Slow' . -v -benchmem -cpuprofile='cpu.out' -memprofile='mem.out' -memprofilerate=1
```

```shell
# Просмотр результатов по процессору в web-интерфейсе
go tool pprof -http=:8080 .\hw3.test.exe .\cpu.out
```

```shell
# Просмотр результатов по памяти в web-интерфейсе
go tool pprof -http=:8080 .\hw3.test.exe .\mem.out
```
## CPU

![img_1.png](img_1.png)

Основные издержки приходятся на json.Unmarshal и regexp.Compile.

![img_2.png](img_2.png)

Из-за них происходит большое количество аллокаций, занимающих процессорное время.

![img_3.png](img_3.png)

Суммарно уходит почти секунда времени на json.Unmarshal

![img_4.png](img_4.png)

Почти секунда на regexp.MatchString

![img_5.png](img_5.png)

Еще раз почти секунда на regexp.MatchString спустя несколько строчек.

## Memory

![img_6.png](img_6.png)

Самые большие издержки по памяти приходятся на regexp.MatchString, json.Unmarshal и ioutil.ReadAll.

![img_7.png](img_7.png)![img_8.png](img_8.png)

Для выполнения regexp.MatchString требуется 40 и 26 МБ памяти

![img_9.png](img_9.png)

На json.Unmarshal 10 МБ

![img_10.png](img_10.png)

На чтение всего файла целиком и последующее разделение на строки 4 МБ

## Резюме

1. Самой большой проблемой, как по памяти, так и по процессору являются regexp'ы (MatchString). В сущности происходит проверка, что в строке browser находится подстрока "Androin" или "MSIE" соответственно. При этом происходят аллокации памяти, и тратится процессорное время. Можно прекомпилировать структуру для регулярного выражения, чтобы не делать это каждый раз в цикле. Но поскольку паттерн для регулярного выражения простой, можно заменить его на strings.Contains, который гораздо лучше:

![img_11.png](img_11.png)

После исправления получили следующие результаты: 

![img_12.png](img_12.png)