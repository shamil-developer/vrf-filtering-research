welcome:
	@printf "%s\n" "$$(tput setaf 2)"
	@printf "%s\n" "       ▒▒▒   ▒▒▒       ███████╗██╗░░░██╗░█████╗░░██████╗██████╗░███╗░░██╗"
	@printf "%s\n" "    ▒▒▒▒▒▒   ▒▒▒▒▒▒    ██╔════╝██║░░░██║██╔══██╗██╔════╝██╔══██╗████╗░██║"
	@printf "%s\n" "    ▒▒▒▒▒▒   ▒▒▒▒▒▒    █████╗░░╚██╗░██╔╝██║░░██║╚█████╗░██║░░██║██╔██╗██║"
	@printf "%s\n" "    ▒▒▒▒▒▒             ██╔══╝░░░╚████╔╝░██║░░██║░╚═══██╗██║░░██║██║╚████║"
	@printf "%s\n" "    ▒▒▒▒▒▒   ▒▒▒▒▒▒    ███████╗░░╚██╔╝░░╚█████╔╝██████╔╝██████╔╝██║░╚███║"
	@printf "%s\n" "    ▒▒▒▒▒▒   ▒▒▒▒▒▒    ╚══════╝░░░╚═╝░░░░╚════╝░╚═════╝░╚═════╝░╚═╝░░╚══╝"
	@printf "%s\n" "       ▒▒▒   ▒▒▒"
	@printf "%s\n" "$$(tput sgr0)"

prepare: welcome
	sudo apt install -y \
    golang-go \
    iproute2 \
    nftables \
    iperf3 \
    tcpdump \
    clang \
    llvm \
    libbpf-dev

run: welcome
	sudo go run cmd/lab/main.go