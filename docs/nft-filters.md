# NFTables filter profiles

Документ фиксирует фильтры для benchmark-сценариев. В `filters.yaml` правила задаются списком `rules`, поэтому один профиль может крутить десятки разных проверок, а не размножать один шаблон.

Главная идея теста: большинство правил подбираются так, чтобы проверять пакет и не совпадать с основным `iperf3`-трафиком, а часть правил через `match_every` совпадает с потоком. Так трафик проходит цепочку, но benchmark видит стоимость разных типов условий.

## Topology distribution

В `filters.yaml` есть блок `topology`. Он задает элементы карты и направления, по которым генератор равномерно раскладывает правила:

| Direction | Segment | iif | oif | Объяснение |
|---|---|---|---|---|
| client-to-bridge | bridge | `veth-c` | `veth-b` | Клиентский край bridge `br0`. |
| bridge-to-vrf | bridge-vrf | `veth-b` | `veth-vrf` | Переход от bridge `br0` к VRF `vrf-blue`. |
| vrf-to-server | vrf | `veth-vrf` | `veth-s` | Участок VRF `vrf-blue` к серверу. |
| server-to-vrf | vrf | `veth-s` | `veth-vrf` | Обратный путь от сервера к VRF. |
| vrf-to-bridge | bridge-vrf | `veth-vrf` | `veth-b` | Обратный переход от VRF к bridge. |
| bridge-to-client | bridge | `veth-b` | `veth-c` | Обратный путь bridge к клиенту. |

Если шаблон правила сам не содержит `iifname` или `oifname`, генератор автоматически добавляет к нему текущий путь: `iifname "{iif}" oifname "{oif}"`. Поэтому каждый шаг benchmark покрывает все сегменты карты, а не кладет правила в одну точку.

## Actions

В `filters.yaml` фильтр и действие разделены. Шаблоны правил используют `{action}`, а конкретное действие берется из блока `actions`. Поэтому benchmark проверяет не только разные match-условия, но и разные nftables statements.

| Action | nft fragment | Объяснение |
|---|---|---|
| counter | `counter` | Только считает пакет. |
| accept | `accept` | Явно пропускает пакет. |
| drop | `drop` | Дропает только синтетический непопадающий трафик; для совпавшего iperf заменяется на `counter`. |
| meta-mark-set | `meta mark set {mark} counter` | Меняет skb mark и считает пакет. |
| log | `log prefix "bench-{index}-{segment}" counter` | Логирует только синтетический непопадающий трафик; для iperf заменяется на `counter`. |
| nftrace | `meta nftrace set 1 counter` | Включает nftrace только на синтетическом непопадающем трафике. |
| queue | `queue num 0` | Отправляет только синтетический непопадающий трафик в queue; для iperf заменяется на `counter`. |
| rate-limit | `limit rate over 1000 mbytes/second counter` | Добавляет limit expression и счетчик. |

## Easy filters

| Name | nft command | Объяснение |
|---|---|---|
| IP источника | `nft add rule inet bench forward ip saddr 198.18.1.10 counter` | Простая проверка IPv4 source address. |
| IP назначения | `nft add rule inet bench forward ip daddr 203.0.113.10 counter` | Простая проверка IPv4 destination address. |
| TCP source port | `nft add rule inet bench forward tcp sport 10010 counter` | Проверка исходного TCP-порта. |
| TCP destination port | `nft add rule inet bench forward tcp dport 10010 counter` | Проверка TCP-порта назначения. |
| UDP source port | `nft add rule inet bench forward udp sport 10010 counter` | Проверка исходного UDP-порта. |
| UDP destination port | `nft add rule inet bench forward udp dport 10010 counter` | Проверка UDP-порта назначения. |
| L4 protocol | `nft add rule inet bench forward ip protocol udp counter` | Проверка протокола транспортного уровня. |
| IP TTL | `nft add rule inet bench forward ip ttl 42 counter` | Проверка TTL в IPv4-заголовке. |
| IP length | `nft add rule inet bench forward ip length 512 counter` | Проверка длины IPv4-пакета. |
| ICMP type | `nft add rule inet bench forward ip protocol icmp icmp type echo-request counter` | Проверка ICMP-типа. |
| Input interface | `nft add rule inet bench forward iifname "veth-c" counter` | Проверка входного интерфейса. |
| Output interface | `nft add rule inet bench forward oifname "veth-c" counter` | Проверка выходного интерфейса. |
| Meta length | `nft add rule inet bench forward meta length 512 counter` | Проверка длины пакета через meta expression. |

## Medium filters

| Name | nft command | Объяснение |
|---|---|---|
| Source and destination IP | `nft add rule inet bench forward ip saddr 198.18.1.10 ip daddr 203.0.113.10 counter` | Комбинация IP источника и назначения. |
| TCP source and destination ports | `nft add rule inet bench forward tcp sport 10010 tcp dport 10011 counter` | Комбинация двух TCP-портов. |
| Input and output interfaces | `nft add rule inet bench forward iifname "veth-c" oifname "veth-s" counter` | Проверка направления через пару интерфейсов. |
| Source IP and TCP port | `nft add rule inet bench forward ip saddr 198.18.1.10 tcp dport 10010 counter` | Частый ACL-сценарий: клиентский IP и порт сервиса. |
| Destination IP and TCP source port | `nft add rule inet bench forward ip daddr 203.0.113.10 tcp sport 10010 counter` | Проверка адреса назначения и source port. |
| Protocol and port | `nft add rule inet bench forward meta l4proto udp udp dport 10010 counter` | Проверка L4-протокола и порта. |
| TCP SYN flag | `nft add rule inet bench forward tcp flags & syn == syn tcp dport 10010 counter` | Проверка SYN-флага TCP. |
| TCP ACK flag | `nft add rule inet bench forward tcp flags & ack == ack tcp dport 10010 counter` | Проверка ACK-флага TCP. |
| TCP FIN or RST flags | `nft add rule inet bench forward tcp flags & (fin\|rst) != 0 tcp dport 10010 counter` | Проверка закрытия или сброса TCP-соединения. |
| Meta mark | `nft add rule inet bench forward meta mark 10 counter` | Проверка skb mark. |
| Meta priority | `nft add rule inet bench forward meta priority 10 counter` | Проверка priority метаданных пакета. |
| Meta skuid | `nft add rule inet bench forward meta skuid 10 counter` | Проверка UID владельца socket. |
| Meta skgid | `nft add rule inet bench forward meta skgid 10 counter` | Проверка GID владельца socket. |
| Limit rate | `nft add rule inet bench forward ip saddr 198.18.1.10 limit rate over 1000 mbytes/second counter` | Проверка rate-limit expression. |
| Random generator | `nft add rule inet bench forward numgen random mod 1000 == 10 ip saddr 198.18.1.10 counter` | Проверка random expression вместе с match-условием. |
| Payload length | `nft add rule inet bench forward meta length 512 ip saddr 198.18.1.10 counter` | Комбинация размера пакета и IP источника. |
| IP fragment offset | `nft add rule inet bench forward ip frag-off 10 counter` | Проверка fragment offset. |
| IP DSCP | `nft add rule inet bench forward ip dscp cs1 counter` | Проверка DSCP-класса. |

## Hard filters

| Name | nft command | Объяснение |
|---|---|---|
| Conntrack state | `nft add rule inet bench forward ct state invalid ip saddr 198.18.1.10 counter` | Stateful-проверка состояния соединения. |
| Conntrack status | `nft add rule inet bench forward ct status dnat ip saddr 198.18.1.10 counter` | Проверка conntrack status. |
| Conntrack direction | `nft add rule inet bench forward ct direction reply ip saddr 198.18.1.10 counter` | Проверка направления conntrack-записи. |
| Conntrack mark | `nft add rule inet bench forward ct mark 10 ip saddr 198.18.1.10 counter` | Проверка conntrack mark. |
| Conntrack zone | `nft add rule inet bench forward ct zone 1010 ip saddr 198.18.1.10 counter` | Проверка conntrack zone. |
| Conntrack original source | `nft add rule inet bench forward ct original ip saddr 198.18.1.10 counter` | Проверка source address из original tuple. |
| Conntrack original destination | `nft add rule inet bench forward ct original ip daddr 203.0.113.10 counter` | Проверка destination address из original tuple. |
| Conntrack reply source | `nft add rule inet bench forward ct reply ip saddr 203.0.113.10 counter` | Проверка source address из reply tuple. |
| Conntrack reply destination | `nft add rule inet bench forward ct reply ip daddr 198.18.1.10 counter` | Проверка destination address из reply tuple. |
| Conntrack original source port | `nft add rule inet bench forward ct original ip saddr 198.18.1.10 tcp sport 10010 counter` | Проверка source port в original tuple-сценарии. |
| Conntrack original destination port | `nft add rule inet bench forward ct original ip daddr 203.0.113.10 tcp dport 10010 counter` | Проверка destination port в original tuple-сценарии. |
| Conntrack reply source port | `nft add rule inet bench forward ct reply ip saddr 203.0.113.10 tcp sport 10010 counter` | Проверка source port в reply tuple-сценарии. |
| Conntrack reply destination port | `nft add rule inet bench forward ct reply ip daddr 198.18.1.10 tcp dport 10010 counter` | Проверка destination port в reply tuple-сценарии. |
| FIB source type | `nft add rule inet bench forward fib saddr type local ip saddr 198.18.1.10 counter` | FIB lookup по source address. |
| FIB destination type | `nft add rule inet bench forward fib daddr type local ip daddr 203.0.113.10 counter` | FIB lookup по destination address. |
| FIB source interface lookup | `nft add rule inet bench forward fib saddr . iif oif missing ip saddr 198.18.1.10 counter` | FIB lookup по source address и входному интерфейсу. |
| FIB destination interface lookup | `nft add rule inet bench forward fib daddr . iif oif missing ip daddr 203.0.113.10 counter` | FIB lookup по destination address и входному интерфейсу. |
| IPsec reqid | `nft add rule inet bench forward ipsec in reqid 200010 ip saddr 198.18.1.10 counter` | Проверка IPsec request id. |
| IPsec SPI | `nft add rule inet bench forward ipsec in spi 100010 ip saddr 198.18.1.10 counter` | Проверка IPsec SPI. |
| Socket transparent | `nft add rule inet bench forward socket transparent 1 ip saddr 198.18.1.10 counter` | Проверка transparent socket expression. |
| Route nexthop | `nft add rule inet bench forward rt ip nexthop 203.0.113.10 ip saddr 198.18.1.10 counter` | Проверка IPv4 route nexthop. |
| Route classid | `nft add rule inet bench forward rt classid 10 ip saddr 198.18.1.10 counter` | Проверка route classid. |
| Route IP nexthop | `nft add rule inet bench forward rt ip nexthop 203.0.113.10 ip saddr 198.18.1.10 counter` | Проверка IPv4 route nexthop. |
| Anonymous set lookup | `nft add rule inet bench forward ip saddr { 198.18.1.10, 198.19.1.10 } counter` | Lookup по anonymous set. |
| Anonymous map lookup | `nft add rule inet bench forward meta mark set ip saddr map { 198.18.1.10 : 10 } counter` | Lookup по anonymous map. |
| Verdict map | `nft add rule inet bench forward ip saddr vmap { 198.18.1.10 : accept }` | Verdict map по IP-адресу. |
| Flowtable marker | `nft add rule inet bench forward ip saddr 198.18.1.10 meta mark set 10 counter` | Маркировка пакета для flowtable-подобного сценария. |
| Queue | `nft add rule inet bench forward ip saddr 198.18.1.10 queue num 0` | Queue action, подобранный так, чтобы основной iperf не попадал в очередь. |
| Log | `nft add rule inet bench forward ip saddr 198.18.1.10 log prefix "bench-nonmatch" counter` | Log statement, подобранный так, чтобы не логировать основной iperf. |
| NFTrace | `nft add rule inet bench forward ip saddr 198.18.1.10 meta nftrace set 1 counter` | Включение nftrace только для синтетического непопадающего IP. |
