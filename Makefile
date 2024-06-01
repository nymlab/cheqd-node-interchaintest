###########################
###   Build contracts   ###
###########################

build-contracts:

	cd contracts; make build; cd ..
	for file in ./contracts/artifacts/*.wasm; do \
        echo "$$file"; \
				mv "$$file" ./contracts_wasm/$$(basename "$$file" -aarch64.wasm).wasm; \
  done
	cp ./contracts/artifacts/** ./contracts_wasm/

.PHONY: build-contracts
