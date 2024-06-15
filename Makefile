###########################
###   Build contracts   ###
###########################

UNAMEP := $(shell uname -p)
build-contracts:
	mkdir -p contracts_wasm
	cd contracts; make build; cd ..
	for file in ./contracts/artifacts/*.wasm; do \
		if [ $(UNAMEP) = aarch64 ]; then \
			echo "$$file"; \
					mv "$$file" ./contracts_wasm/$$(basename "$$file" -aarch64.wasm).wasm; \
		else \
			echo "$$file"; \
					mv "$$file" ./contracts_wasm; \
		fi \
	done
	cp ./contracts/artifacts/** ./contracts_wasm/

.PHONY: build-contracts
