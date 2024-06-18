###########################
###   Build contracts   ###
###########################

#!/bin/bash

UNAMEP := $(shell uname -p)
build-contracts:
	echo $(UNAMEP)
	mkdir -p contracts_wasm
	cd contracts; make build; cd ..
	for file in ./contracts/artifacts/*; do \
		if [ $(UNAMEP) = arm ]; then \
			echo "$$file"; \
		    BASENAME=$$(basename $$file); \
		    NEWNAME=$$(echo $$BASENAME | sed 's/-aarch64//'); \
		    mv $$file ./contracts_wasm/$$NEWNAME; \
		else \
			echo "$$file"; \
					mv "$$file" ./contracts_wasm; \
		fi \
	done

.PHONY: build-contracts
