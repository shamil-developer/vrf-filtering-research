# Создание базовой сетевой изоляции

## 1. Создаём namespace

```bash
ip netns add ns1
ip netns add ns2
```

> `netns` - Network Namespace
> `ns` - Namespace

Проверяем:

```bash
ip netns list
```

Ожидаем:
- `ns1`
- `ns2`

## 2. Создаём виртуальный кабель

```bash
ip link add veth1 type veth peer name veth2
```

> `veth1` - Virtual Ethernet
> peer - сосед, партнёр, другая сторона соединения

Получили:

```txt
veth1 <----> veth2
```

## 3. Засовываем концы кабеля

```bash
ip link set veth1 netns ns1
ip link set veth2 netns ns2
```

Теперь:

```txt
ns1
 └─ veth1
ns2
 └─ veth2
```

## 4. Назначаем IP

```bash
ip netns exec ns1 ip addr add 10.0.0.1/24 dev veth1
ip netns exec ns2 ip addr add 10.0.0.2/24 dev veth2
```

> `dev` - device

## 5. Поднимаем loopback

```bash
ip netns exec ns1 ip link set lo up
ip netns exec ns2 ip link set lo up
```

## 6. Поднимаем интерфейсы

```bash
ip netns exec ns1 ip link set veth1 up
ip netns exec ns2 ip link set veth2 up
```

## 7. Проверяем

```bash
ip netns exec ns1 ip addr
```

```bash
ip netns exec ns2 ip addr
```

## 8. Пинг

```bash
ip netns exec ns1 ping 10.0.0.2
```


```txt
veth1 | 10.0.0.1/24 <----> veth2 | 10.0.0.2/24
```

# Создание L2 сегмента через Bridge

## 1. Создаём bridge

```bash
ip link add br0 type bridge
```

Проверяем:

```bash
ip link show br0
```

## 2. Поднимаем bridge

```bash
ip link set br0 up
```

## 3. Удаляем старую схему

Сначала удаляем старый кабель между namespace.

```bash
ip netns exec ns1 ip link delete veth1
```

Пара удалится целиком:

```txt
veth1 <----> veth2
```

## 4. Создаём новые пары

Для host1:

```bash
ip link add veth1 type veth peer name br-veth1
```

Для host2:

```bash
ip link add veth2 type veth peer name br-veth2
```

Получаем:

```txt
veth1 <----> br-veth1
veth2 <----> br-veth2
```

## 5. Перемещаем интерфейсы в namespace

```bash
ip link set veth1 netns ns1
ip link set veth2 netns ns2
```

Получаем:

```txt
ns1
 └─ veth1
ns2
 └─ veth2
host
 ├─ br-veth1
 └─ br-veth2
```

## 6. Подключаем к bridge

```bash
ip link set br-veth1 master br0
ip link set br-veth2 master br0
```

## 7. Поднимаем интерфейсы

На host:

```bash
ip link set br-veth1 up
ip link set br-veth2 up
```

В namespace:

```bash
ip netns exec ns1 ip link set lo up
ip netns exec ns1 ip link set veth1 up
ip netns exec ns2 ip link set lo up
ip netns exec ns2 ip link set veth2 up
```

## 8. Назначаем адреса

```bash
ip netns exec ns1 ip addr add 10.0.0.1/24 dev veth1
ip netns exec ns2 ip addr add 10.0.0.2/24 dev veth2
```

## 9. Проверяем

```bash
ip netns exec ns1 ping 10.0.0.2
```

```bash
ip netns exec ns2 ping 10.0.0.1
```

Посмотреть, что bridge выучил MAC-адреса

```bash
bridge fdb show
```
> `fdb` - Forwarding Database

Вывод будет примерно такой
```log
...
e2:4f:6d:21:92:dd dev br-veth2 master br0
...
```
> master (главный) - то есть кому пренодлежит этот фход

Схема получится такая:

```txt
ns1
 |
veth1 (10.0.0.1/24)
 |
br-veth1
 |
+----- br0 -----+
 |
br-veth2
 |
veth2 (10.0.0.2/24)
 |
ns2
```

## # Проверка nftables на Bridge

1. Убедись, что пинг сейчас работает

```bash
ip netns exec ns1 ping -c 3 10.0.0.2
```

## 2. Создаём таблицу для bridge-трафика

```bash
nft add table bridge br_test
```

`bridge` тут значит: правила будут смотреть трафик, который проходит через Linux Bridge.

## 3. Создаём цепочку `forward`

```bash
nft add chain bridge br_test forward '{ type filter hook forward priority 0; policy accept; }'
```

## 4. Добавляем правило-счётчик

```bash
nft add rule bridge br_test forward counter
```

## 5. Пингуем

```bash
ip netns exec ns1 ping -c 5 10.0.0.2
```

## 6. Смотрим счётчик

```bash
nft list ruleset  | cat
```

Ищи строку типа:

```log
counter packets 5 bytes ...
```

## 7. Теперь проверяем DROP

```bash
nft add rule bridge br_test forward drop
```

Пингуем снова:

```bash
ip netns exec ns1 ping -c 3 10.0.0.2
```

Ожидаем: пинг сломался.

Это доказывает: nftables реально может фильтровать трафик на bridge.

# Проверка привязки правил nftables к интерфейсам Bridge

## 1. Смотрим текущие правила

```
nft -a list table bridge br_test | cat
```
> `-a` — показать handle (ID правила).


## 2. Удаляем правила прошлых экспериментов

Удаляем counter:

```bash
nft delete rule bridge br_test forward handle 5
```

Удаляем drop:

```bash
nft delete rule bridge br_test forward handle 6
```

Проверяем:

```bash
nft -a list table bridge br_test
```

Должно остаться:

```bat
table bridge br_test {
    chain forward {
        type filter hook forward priority filter; policy accept;
    }
}
```

## 3. Добавляем счётчик по входному интерфейсу

```bash
nft add rule bridge br_test forward iifname "br-veth1" counter
```

> `iifname` - Input Interface Name

## 4. Проверяем правило

```bash
nft -a list table bridge br_test
```

## 5. Генерируем трафик

```bash
ip netns exec ns1 ping -c 5 10.0.0.2
```

## 6. Проверяем счётчик

```bash
nft list table bridge br_test | cat
```

## 7. Проверяем блокировку по интерфейсу

Добавляем новое правило:

```bash
nft add rule bridge br_test forward iifname "br-veth1" drop
```

## 8. Проверяем связность

```bash
ip netns exec ns1 ping -c 5 10.0.0.2
```

## 9. Проверяем обратное направление

```bash
ip netns exec ns2 ping -c 5 10.0.0.1
```

# Проверка фильтрации по MAC-адресу

## 1. Удаляем правила прошлого эксперимента

Смотрим правила:

```bash
nft -a list table bridge br_test
```

Удаляем все старые counter и drop через их handle.

Проверяем:

```bash
nft -a list table bridge br_test
```

Должно остаться:

```js
table bridge br_test {
    chain forward {
        type filter hook forward priority filter; policy accept;
    }
}
```

## 2. Узнаём MAC-адреса интерфейсов

Для ns1:

```bash
ip netns exec ns1 ip link show veth1
```

Получим примерно:

```bash
f2:ee:b8:f9:9d:99
```

Для ns2:

```bash
ip netns exec ns2 ip link show veth2
```

Получим примерно:

```bash
e2:4f:6d:21:92:dd
```

## 3. Добавляем счётчик по MAC источника

Например для ns1:

```bash
nft add rule bridge br_test forward ether saddr f2:ee:b8:f9:9d:99 counter
```

> `saddr` — Source Address.

MAC-адрес источника кадра.

## 4. Генерируем трафик

```bash
ip netns exec ns1 ping -c 5 10.0.0.2
```

## 5. Проверяем счётчик

```bash
nft list table bridge br_test | cat
```

Ожидаем:

```bash
ether saddr f2:ee:b8:f9:9d:99 counter packets > 0
```

## 6. Проверяем блокировку по MAC

Добавляем правило:

```bash
nft add rule bridge br_test forward ether saddr f2:ee:b8:f9:9d:99 drop
```

## 7. Проверяем связность

```bash
ip netns exec ns1 ping -c 5 10.0.0.2
```

Ожидаем:

100% packet loss


## 8. Проверяем второй namespace

```bash
ip netns exec ns2 ping -c 5 10.0.0.1
```

# Проверка фильтрации по Bridge-интерфейсу

## 1. Удаляем правила прошлого эксперимента

Смотрим:

```bash
nft -a list table bridge br_test
```

Удаляем все старые правила через их handle.

Проверяем:

```bash
nft -a list table bridge br_test
```

Должно остаться:

```js
table bridge br_test {
    chain forward {
        type filter hook forward priority filter; policy accept;
    }
}
```

## 2. Добавляем счётчик для bridge

Добавляем правило:

```bash
nft add rule bridge br_test forward meta ibrname "br0" counter
```

meta — метаданные пакета.

> `bri_iifname` — имя bridge, через который кадр вошёл в обработку.

## 3. Генерируем трафик

```bash
ip netns exec ns1 ping -c 5 10.0.0.2
```

## 4. Проверяем счётчик

```bash
nft list table bridge br_test | cat
```

## 5. Проверяем блокировку

Добавляем правило:

```bash
nft add rule bridge br_test forward meta ibrname "br0" drop
```

## 6. Проверяем связность

```bash
ip netns exec ns1 ping -c 5 10.0.0.2
```

## 7. Вывод

Если правило сработало:

```bash
meta bri_iifname "br0"
```

# Проверка фильтрации по VLAN

## 1. Создаём VLAN-aware bridge

Проверяем:

```bash
ip -d link show br0
```

Если VLAN filtering выключен, включаем:

```bash
ip link set br0 type bridge vlan_filtering 1
```

Проверяем:

```bash
ip -d link show br0
```

Ожидаем:

```bash
vlan_filtering 1
```

## 2. Назначаем VLAN портам

Например:

```bash
bridge vlan add dev br-veth1 vid 10 pvid untagged
bridge vlan add dev br-veth2 vid 10 pvid untagged
```

> VID = VLAN Identifier
> PVID = Port VLAN ID
> Untagged = без тега VLAN
> 
Проверяем:

```bash
bridge vlan show
```

## 3. Проверяем связность

```bash
ip netns exec ns1 ping -c 5 10.0.0.2
```

Трафик должен ходить.

## 4. Добавляем счётчик по VLAN

Например:

```bash
nft add rule bridge br_test forward vlan id 10 counter
```

## 5. Генерируем трафик

```bash
ip netns exec ns1 ping -c 5 10.0.0.2
```

## 6. Проверяем счётчик

```bash
nft list table bridge br_test | cat
```

Ожидаем:

```bash
vlan id 10 counter packets > 0
```

## 7. Проверяем DROP

Добавляем:

```bash
nft add rule bridge br_test forward vlan id 10 drop
```

## 8. Проверяем связность

```bash
ip netns exec ns1 ping -c 5 10.0.0.2
```