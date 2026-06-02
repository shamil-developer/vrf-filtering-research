# Создание первого VRF

## 1. Создаём VRF

```bash
ip link add vrf-blue type vrf table 100
```

> vrf-blue — имя VRF.

> table 100 — таблица маршрутизации, которую будет использовать этот VRF.

⸻

## 2. Поднимаем VRF

```bash
ip link set vrf-blue up
```

⸻

## 3. Проверяем интерфейс

```bash
ip link show type vrf
```

Ожидаем примерно:

vrf-blue@NONE: <UP,LOWER_UP>

⸻

## 4. Проверяем таблицу маршрутизации

```bash
ip route show table 100
```

Пока будет пусто.

Это нормально.

⸻

## 5. Проверяем правила Linux

```bash
ip rule
```

# Подключить интерфейсы к VRF

Сейчас у нас есть только:

vrf-blue

Это как роутер без портов.

## 1. Создаём интерфейс

Для начала создадим простой интерфейс:

```bash
ip link add dummy0 type dummy
```

## 2. Проверяем

```bash
ip link show dummy0
```

## 3. Подключаем к VRF

```bash
ip link set dummy0 master vrf-blue
```

Смысл:

dummy0 теперь принадлежит vrf-blue

## 4. Проверяем

```bash
ip link show dummy0
```

Ожидаем примерно:

dummy0@NONE: <...>
master vrf-blue

## 5. Смотрим всех участников VRF

```bash
ip link
```

Ищем:

vrf-blue
dummy0

# Проверить прохождение трафика через VRF

## 1. Создать namespace клиента

```bash
ip netns add ns1
```

## 2. Создать veth-пару

```bash
ip link add veth-host type veth peer name veth-ns
```

Это как виртуальный провод:

veth-host <=====> veth-ns

## 3. Один конец подключить к VRF

```bash
ip link set veth-host master vrf-blue
ip link set veth-host up
```

## 4. Второй конец засунуть в namespace

```bash
ip link set veth-ns netns ns1
```

## 5. Назначить IP

На стороне VRF:

```bash
ip addr add 10.0.0.1/24 dev veth-host
```

Внутри ns:

```bash
ip netns exec ns1 ip addr add 10.0.0.2/24 dev veth-ns
ip netns exec ns1 ip link set lo up
ip netns exec ns1 ip link set veth-ns up
```

## 6. Проверить

Из namespace:

```bash
ip netns exec ns1 ping 10.0.0.1
```

Схема получится:

ns1
  |
10.0.0.2
  |
veth-ns
=================
veth-host
  |
10.0.0.1
  |
vrf-blue

# Проверить counter для VRF

## 1. Создаём таблицу nftables

```bash
nft add table inet test
```

---

## 2. Создаём chain

```bash
nft add chain inet test input '{ type filter hook input priority 0; }'
```

---

## 3. Вешаем counter на VRF

```bash
nft add rule inet test input iifname "vrf-blue" counter
```

---

## 4. Проверяем правило

```bash
nft list ruleset
```

Ожидаем:

```text
table inet test {
    chain input {
        iifname "vrf-blue" counter packets 0 bytes 0
    }
}
```

---

## 5. Генерируем трафик через VRF

Из namespace:

```bash
ip netns exec ns1 ping 10.0.0.1 -c 5
```

---

## 6. Проверяем счётчик

```bash
nft list ruleset
```

Ожидаем:

```text
iifname "vrf-blue" counter packets X bytes Y
```

где:

```text
X > 0
Y > 0
```

---

## Результат

Если счётчик увеличился:

```text
✓ nftables видит трафик VRF

✓ правила можно привязывать к VRF

✓ counter для VRF работает
```

# Проверить drop для VRF

## 1. Удаляем старое правило counter

Смотрим handle:

```bash
nft -a list table inet test
```

Удаляем правило:

```bash
nft delete rule inet test input handle <HANDLE>
```

---

## 2. Создаём правило drop для VRF

```bash
nft add rule inet test input iifname "vrf-blue" drop
```

---

## 3. Проверяем

```bash
nft list table inet test
```

Ожидаем:

```text
table inet test {
    chain input {
        type filter hook input priority filter; policy accept;
        iifname "vrf-blue" drop
    }
}
```

---

## 4. Проверяем связность

Из namespace:

```bash
ip netns exec ns1 ping 10.0.0.1
```

Ожидаем:

```text
Destination Host Unreachable
```

или

```text
100% packet loss
```

---

## 5. Убеждаемся что правило виновато

Удаляем правило:

```bash
nft flush chain inet test input
```

---

## 6. Проверяем ещё раз

```bash
ip netns exec ns1 ping 10.0.0.1
```

Ожидаем успешный ответ:

```text
64 bytes from 10.0.0.1
```

---

## Результат

Если после добавления правила:

```text
ping не проходит
```

а после удаления:

```text
ping снова проходит
```

значит:

```text
✓ nftables видит VRF

✓ drop для VRF работает

✓ трафик через VRF можно блокировать правилами nftables
```

# Проверить привязку правил к VRF-интерфейсу

Цель:

Понять применяется ли правило именно к VRF (`vrf-blue`)
или к интерфейсу внутри VRF (`veth-host`).

---

## 1. Очистить старые правила

```bash
nft flush table inet test
```

---

## 2. Создать правило на VRF

```bash
nft add chain inet test input '{ type filter hook input priority 0; }'

nft add rule inet test input iifname "vrf-blue" counter
```

---

## 3. Сгенерировать трафик

```bash
ip netns exec ns1 ping 10.0.0.1 -c 5
```

---

## 4. Проверить счётчик

```bash
nft list table inet test
```

Записать результат.

---

## 5. Очистить правило

```bash
nft flush chain inet test input
```

---

## 6. Создать правило на интерфейс VRF

```bash
nft add rule inet test input iifname "veth-host" counter
```

---

## 7. Сгенерировать трафик

```bash
ip netns exec ns1 ping 10.0.0.1 -c 5
```

---

## 8. Проверить счётчик

```bash
nft list table inet test
```

---

## Результат

Сравнить:

```text
iifname "vrf-blue"
```

и

```text
iifname "veth-host"
```

Если оба счётчика растут:

```text
✓ nftables умеет матчить VRF

✓ nftables умеет матчить интерфейс внутри VRF
```

Если растёт только:

```text
iifname "veth-host"
```

то:

```text
✓ трафик обрабатывается через интерфейс

✗ матчинг по VRF фактически не используется
```

---

Вывод исследования:

Определить что является правильной точкой привязки правил:

```text
VRF

или

интерфейс внутри VRF
```