APP_NAME := game
OUT_DIR := docs
WEB_DIR := web

.PHONY: wasm clean serve

wasm:
	@mkdir -p $(OUT_DIR)
	GOOS=js GOARCH=wasm go build -o $(OUT_DIR)/$(APP_NAME).wasm ./cmd/game
	@cp $(WEB_DIR)/index.html $(OUT_DIR)/index.html
	@cp $(WEB_DIR)/wasm_exec.js $(OUT_DIR)/wasm_exec.js

clean:
	@rm -rf $(OUT_DIR)

serve: wasm
	@echo "open http://localhost:8080"
	@cd $(OUT_DIR) && python3 -m http.server 8080
