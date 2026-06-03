package ebpf

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/shamil-developer/vrf-filtering-research/internal/filterprofile"
)

var (
	ipSaddrPattern  = regexp.MustCompile(`ip saddr ([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`)
	ipDaddrPattern  = regexp.MustCompile(`ip daddr ([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`)
	tcpSportPattern = regexp.MustCompile(`tcp sport ([0-9]+)`)
	tcpDportPattern = regexp.MustCompile(`tcp dport ([0-9]+)`)
	udpSportPattern = regexp.MustCompile(`udp sport ([0-9]+)`)
	udpDportPattern = regexp.MustCompile(`udp dport ([0-9]+)`)
	ttlPattern      = regexp.MustCompile(`ip ttl ([0-9]+)`)
	ipLengthPattern = regexp.MustCompile(`ip length ([0-9]+)`)
	metaLenPattern  = regexp.MustCompile(`meta length ([0-9]+)`)
	markPattern     = regexp.MustCompile(`(?:meta|ct) mark ([0-9]+)`)
)

func renderProgram(
	rules []filterprofile.GeneratedRule,
	ifIndexes map[string]int,
) string {
	var body strings.Builder
	for _, rule := range rules {
		if rule.IsFinal {
			continue
		}
		iface := iifFromRule(rule)
		ifIndex, ok := ifIndexes[iface]
		if !ok {
			continue
		}

		body.WriteString("    if (")
		body.WriteString(renderCondition(rule.Command, ifIndex))
		body.WriteString(") {\n")
		body.WriteString(renderAction(rule))
		body.WriteString("    }\n")
	}

	return fmt.Sprintf(`#include <linux/bpf.h>
#include <linux/pkt_cls.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <linux/icmp.h>
#include <linux/in.h>
#include <bpf/bpf_helpers.h>

#define BPF_HTONS(x) ((__u16)__builtin_bswap16((__u16)(x)))
#ifndef IPPROTO_TCP
#define IPPROTO_TCP 6
#endif
#ifndef IPPROTO_UDP
#define IPPROTO_UDP 17
#endif
#ifndef IPPROTO_ICMP
#define IPPROTO_ICMP 1
#endif
#define IP4(a, b, c, d) ((__u32)(((__u32)(d) << 24) | ((__u32)(c) << 16) | ((__u32)(b) << 8) | ((__u32)(a))))

struct packet {
    struct iphdr *ip;
    void *l4;
    void *data_end;
};

static __always_inline int parse_packet(struct __sk_buff *skb, struct packet *pkt) {
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end) {
        return 0;
    }
    if (eth->h_proto != BPF_HTONS(ETH_P_IP)) {
        return 0;
    }

    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end) {
        return 0;
    }
    pkt->ip = ip;
    pkt->l4 = (void *)ip + (ip->ihl * 4);
    pkt->data_end = data_end;
    return 1;
}

static __always_inline int tcp_sport(struct packet *pkt, __u16 port) {
    if (pkt->ip->protocol != IPPROTO_TCP) {
        return 0;
    }
    struct tcphdr *tcp = pkt->l4;
    if ((void *)(tcp + 1) > pkt->data_end) {
        return 0;
    }
    return tcp->source == BPF_HTONS(port);
}

static __always_inline int tcp_dport(struct packet *pkt, __u16 port) {
    if (pkt->ip->protocol != IPPROTO_TCP) {
        return 0;
    }
    struct tcphdr *tcp = pkt->l4;
    if ((void *)(tcp + 1) > pkt->data_end) {
        return 0;
    }
    return tcp->dest == BPF_HTONS(port);
}

static __always_inline int udp_sport(struct packet *pkt, __u16 port) {
    if (pkt->ip->protocol != IPPROTO_UDP) {
        return 0;
    }
    struct udphdr *udp = pkt->l4;
    if ((void *)(udp + 1) > pkt->data_end) {
        return 0;
    }
    return udp->source == BPF_HTONS(port);
}

static __always_inline int udp_dport(struct packet *pkt, __u16 port) {
    if (pkt->ip->protocol != IPPROTO_UDP) {
        return 0;
    }
    struct udphdr *udp = pkt->l4;
    if ((void *)(udp + 1) > pkt->data_end) {
        return 0;
    }
    return udp->dest == BPF_HTONS(port);
}

static __always_inline int tcp_flags_any(struct packet *pkt, __u8 mask) {
    if (pkt->ip->protocol != IPPROTO_TCP) {
        return 0;
    }
    struct tcphdr *tcp = pkt->l4;
    if ((void *)(tcp + 1) > pkt->data_end) {
        return 0;
    }
    return (((__u8 *)tcp)[13] & mask) != 0;
}

SEC("classifier")
int bench_classifier(struct __sk_buff *skb) {
    struct packet pkt = {};
    if (!parse_packet(skb, &pkt)) {
        return TC_ACT_OK;
    }

%s
    return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";
`, body.String())
}

func renderCondition(
	command string,
	ifIndex int,
) string {
	conditions := []string{
		fmt.Sprintf("skb->ifindex == %d", ifIndex),
	}

	for _, ip := range ipSaddrPattern.FindAllStringSubmatch(command, -1) {
		conditions = append(conditions, fmt.Sprintf("pkt.ip->saddr == %s", ipLiteral(ip[1])))
	}
	for _, ip := range ipDaddrPattern.FindAllStringSubmatch(command, -1) {
		conditions = append(conditions, fmt.Sprintf("pkt.ip->daddr == %s", ipLiteral(ip[1])))
	}
	for _, match := range tcpSportPattern.FindAllStringSubmatch(command, -1) {
		conditions = append(conditions, fmt.Sprintf("tcp_sport(&pkt, %s)", match[1]))
	}
	for _, match := range tcpDportPattern.FindAllStringSubmatch(command, -1) {
		conditions = append(conditions, fmt.Sprintf("tcp_dport(&pkt, %s)", match[1]))
	}
	for _, match := range udpSportPattern.FindAllStringSubmatch(command, -1) {
		conditions = append(conditions, fmt.Sprintf("udp_sport(&pkt, %s)", match[1]))
	}
	for _, match := range udpDportPattern.FindAllStringSubmatch(command, -1) {
		conditions = append(conditions, fmt.Sprintf("udp_dport(&pkt, %s)", match[1]))
	}
	if strings.Contains(command, "ip protocol udp") || strings.Contains(command, "meta l4proto udp") {
		conditions = append(conditions, "pkt.ip->protocol == IPPROTO_UDP")
	}
	if strings.Contains(command, "ip protocol tcp") || strings.Contains(command, "meta l4proto tcp") {
		conditions = append(conditions, "pkt.ip->protocol == IPPROTO_TCP")
	}
	if strings.Contains(command, "ip protocol icmp") {
		conditions = append(conditions, "pkt.ip->protocol == IPPROTO_ICMP")
	}
	for _, match := range ttlPattern.FindAllStringSubmatch(command, -1) {
		conditions = append(conditions, fmt.Sprintf("pkt.ip->ttl == %s", match[1]))
	}
	for _, match := range ipLengthPattern.FindAllStringSubmatch(command, -1) {
		conditions = append(conditions, fmt.Sprintf("pkt.ip->tot_len == BPF_HTONS(%s)", match[1]))
	}
	for _, match := range metaLenPattern.FindAllStringSubmatch(command, -1) {
		conditions = append(conditions, fmt.Sprintf("skb->len == %s", match[1]))
	}
	for _, match := range markPattern.FindAllStringSubmatch(command, -1) {
		conditions = append(conditions, fmt.Sprintf("skb->mark == %s", match[1]))
	}
	if strings.Contains(command, "ip dscp cs1") {
		conditions = append(conditions, "(pkt.ip->tos & 0xfc) == 0x20")
	}
	if strings.Contains(command, "icmp type echo-request") {
		conditions = append(conditions, "pkt.ip->protocol == IPPROTO_ICMP")
	}
	if strings.Contains(command, "tcp flags & syn") {
		conditions = append(conditions, "tcp_flags_any(&pkt, 0x02)")
	}
	if strings.Contains(command, "tcp flags & ack") {
		conditions = append(conditions, "tcp_flags_any(&pkt, 0x10)")
	}
	if strings.Contains(command, "tcp flags & (fin|rst)") {
		conditions = append(conditions, "tcp_flags_any(&pkt, 0x05)")
	}
	if strings.Contains(command, "numgen random") {
		conditions = append(conditions, "(skb->len % 1000) == 0")
	}
	if strings.Contains(command, "ip frag-off") {
		conditions = append(conditions, "(pkt.ip->frag_off & BPF_HTONS(0x1fff)) != 0")
	}

	return strings.Join(conditions, " && ")
}

func renderAction(
	rule filterprofile.GeneratedRule,
) string {
	switch actionFromRule(rule) {
	case "accept":
		return "        return TC_ACT_OK;\n"
	case "drop", "queue":
		return "        return TC_ACT_SHOT;\n"
	case "meta-mark-set":
		return fmt.Sprintf("        skb->mark = %d;\n", rule.Index%1024)
	case "log", "nftrace", "rate-limit", "counter":
		return "        ;\n"
	default:
		return "        ;\n"
	}
}

func actionFromRule(
	rule filterprofile.GeneratedRule,
) string {
	command := rule.Command
	switch {
	case strings.Contains(command, " queue "):
		return "queue"
	case strings.Contains(command, " drop"):
		return "drop"
	case strings.Contains(command, " accept"):
		return "accept"
	case strings.Contains(command, "meta mark set"):
		return "meta-mark-set"
	case strings.Contains(command, "log prefix"):
		return "log"
	case strings.Contains(command, "meta nftrace set"):
		return "nftrace"
	case strings.Contains(command, "limit rate"):
		return "rate-limit"
	case strings.Contains(command, " counter"):
		return "counter"
	default:
		return rule.Action
	}
}

func ipLiteral(
	value string,
) string {
	parts := strings.Split(value, ".")
	if len(parts) != 4 {
		return "0"
	}
	numbers := make([]int, 4)
	for index, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return "0"
		}
		numbers[index] = number
	}

	return fmt.Sprintf("IP4(%d, %d, %d, %d)", numbers[0], numbers[1], numbers[2], numbers[3])
}
