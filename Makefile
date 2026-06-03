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

lab: welcome
	docker compose build lab
	docker compose run --rm lab

bench-nft: welcome
	docker compose build lab
	docker compose run --rm -e LAB_ENGINE=nft -e LAB_NAME=nft lab

bench-tc-ebpf: welcome
	docker compose build lab
	docker compose run --rm -e LAB_ENGINE=tc+ebpf -e LAB_NAME=tc_ebpf lab

bench-ebpf: bench-tc-ebpf

compare: welcome
	docker compose build lab
	docker compose run --rm -e LAB_CONFIG=compare.yaml lab

full-compare: welcome
	@mkdir -p log
	@log_file="log/$$(date +%Y-%m-%d_%H-%M-%S)_full-compare.log"; \
	printf "Полный лог запуска: %s\n" "$$log_file"; \
	bash -o pipefail -c ' \
		docker compose build lab && \
		docker compose run --rm -e LAB_ENGINE=nft -e LAB_NAME=nft lab && \
		docker compose run --rm -e LAB_ENGINE=tc+ebpf -e LAB_NAME=tc_ebpf lab && \
		docker compose run --rm -e LAB_CONFIG=compare.yaml lab \
	' 2>&1 | tee "$$log_file"

docker-build: welcome
	docker compose build

docker-run: welcome
	docker compose run --rm lab

docker-lab: welcome
	docker compose run --rm --entrypoint /bin/bash lab -c "go run ./cmd/lab && exec /bin/bash"

docker-lab-shell: welcome
	docker compose run --rm --entrypoint /bin/bash lab

docker-shell: welcome
	docker compose run --rm ubuntu
