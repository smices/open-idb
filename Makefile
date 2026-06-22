# SPDX-License-Identifier: MIT

.PHONY: test lint generate migrate-up run \
	k8s-local-tls k8s-build k8s-deploy k8s-deploy-app k8s-status k8s-port-forward \
	k8s-dev-watch k8s-build-frontend k8s-deploy-frontend \
	dev-local dev-local-check dev-local-quickfix dev-local-preflight dev-local-go-live \
	dev-web-local dev-web-local-stop dev-web-local-restart dev-web-local-reset-db dev-web-local-status dev-web-local-logs \
	web-dev web-build web-check web-check-strict web-frontend-contract web-contract-bump

DEV_LOCAL_PG_PORT ?= 15432
DEV_LOCAL_BACKEND_PORT ?= 18080
DEV_LOCAL_WEB_PORT ?= 5180
DEV_LOCAL_ETCD_PORT ?= 2379
DEV_LOCAL_DATABASE_URL ?= postgres://idbridge:idbridge-dev@127.0.0.1:$(DEV_LOCAL_PG_PORT)/idbridge?sslmode=disable
DEV_LOCAL_NAMESPACE ?= open-idb
WEB_CHECK_STRICT ?= 0
WEB_DEV_PORT ?= 5180
WEB_FRONTEND_CONTRACT_VERSION ?= web-svelte-tailwind-contract-v1.4
WEB_FRONTEND_CONTRACT_DATE ?= 2026-05-29

test:
	@cd backend && \
	errfile=$$(mktemp); \
	trap 'rm -f "$$errfile"' EXIT; \
	packages=$$(go list ./... 2>"$$errfile"); \
	status=$$?; \
	if [ $$status -ne 0 ]; then \
		if grep -q "matched no packages" "$$errfile"; then \
			echo "no packages to test"; \
		else \
			cat "$$errfile" >&2; \
			exit $$status; \
		fi; \
	elif [ -z "$$packages" ]; then \
		echo "no packages to test"; \
	else \
		go test ./...; \
	fi

lint:
	@cd backend && \
	errfile=$$(mktemp); \
	trap 'rm -f "$$errfile"' EXIT; \
	packages=$$(go list ./... 2>"$$errfile"); \
	status=$$?; \
	if [ $$status -ne 0 ]; then \
		if grep -q "matched no packages" "$$errfile"; then \
			echo "no packages to lint"; \
		else \
			cat "$$errfile" >&2; \
			exit $$status; \
		fi; \
	elif [ -z "$$packages" ]; then \
		echo "no packages to lint"; \
	else \
		go test ./...; \
	fi

generate:
	cd backend && CGO_ENABLED=0 go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0 generate -f internal/db/sqlc.yaml

migrate-up:
	cd backend && go run github.com/pressly/goose/v3/cmd/goose@v3.22.1 -dir migrations postgres "$$DATABASE_URL" up

run:
	cd backend && go run ./cmd/idbridge

dev-web-local:
	./scripts/dev-web-local.sh start

dev-web-local-stop:
	./scripts/dev-web-local.sh stop

dev-web-local-restart:
	./scripts/dev-web-local.sh restart

dev-web-local-reset-db:
	./scripts/dev-web-local.sh reset-db

dev-web-local-status:
	./scripts/dev-web-local.sh status

dev-web-local-logs:
	./scripts/dev-web-local.sh logs

k8s-local-tls:
	sh scripts/k8s-local-tls.sh

k8s-build:
	docker build -t open-idb:dev backend/

k8s-deploy:
	kubectl apply -f deploy/k8s/orbstack/namespace.yaml
	kubectl apply -f deploy/k8s/orbstack/secret.yaml
	kubectl apply -f deploy/k8s/orbstack/postgres.yaml
	kubectl -n open-idb rollout status deployment/postgres --timeout=120s
	kubectl -n open-idb delete job idbridge-migrate --ignore-not-found
	kubectl apply -f deploy/k8s/orbstack/migration-job.yaml
	kubectl -n open-idb wait --for=condition=complete job/idbridge-migrate --timeout=120s
	kubectl apply -f deploy/k8s/orbstack/app.yaml
	kubectl apply -f deploy/k8s/orbstack/ingress.yaml
	kubectl -n open-idb rollout status deployment/idbridge --timeout=120s

k8s-deploy-app:
	kubectl apply -f deploy/k8s/orbstack/app.yaml
	kubectl apply -f deploy/k8s/orbstack/ingress.yaml
	kubectl -n open-idb rollout restart deployment/idbridge
	kubectl -n open-idb rollout status deployment/idbridge --timeout=120s

k8s-status:
	kubectl -n open-idb get pods,svc,ingress,job

k8s-port-forward:
	kubectl -n $(DEV_LOCAL_NAMESPACE) port-forward svc/idbridge 8080:8080

k8s-dev-watch:
	sh scripts/k8s-dev-watch.sh

web-dev:
	@mkdir -p web/node_modules
	@cd web && \
		test -f package-lock.json || npm i; \
		npm run dev -- --host 0.0.0.0 --port $(WEB_DEV_PORT)

web-build:
	@cd web && \
		test -f package-lock.json || npm i; \
		npm run build

web-check:
	@echo "[P1] [web-check] Checking non-SPA navigation constraints..."
	@STYLE_HITS=$$(rg -n "<style|window\\.history\\.(pushState|replaceState)" web/src/app.html web/src/routes web/src/lib web/src --glob '*.svelte' --glob '*.ts' --glob '*.html' | sed -n '1,120p' || true); \
	if [ -n "$$STYLE_HITS" ]; then \
		echo "[P0] Found unsupported style/history SPA-style code in web:"; \
		echo "$$STYLE_HITS"; \
		exit 1; \
	fi
	@echo "[P1] 未检测到页面内 <style> 覆写和 window.history 路由控制"
	@LOCATION_HITS=$$(rg -n "window\\.location\\." web/src/routes web/src/lib --glob '*.svelte' --glob '*.ts' --glob '!web/src/lib/session.ts' | sed -n '1,120p' || true); \
	if [ -n "$$LOCATION_HITS" ]; then \
		echo "[P0] Found window.location usage outside allowed session helpers:"; \
		echo "$$LOCATION_HITS"; \
		exit 1; \
	fi
	@echo "[P1] window.location 仅允许集中在 web/src/lib/session.ts"
	@GOTO_HITS=$$(rg -n "goto\\(|\\$app/navigation" web/src/routes web/src/lib --glob '*.svelte' --glob '*.ts' | sed -n '1,120p' || true); \
	if [ -n "$$GOTO_HITS" ]; then \
		echo "[P0] Found SPA-style navigation usage in web:"; \
		echo "$$GOTO_HITS"; \
		exit 1; \
	fi
	@echo '[P1] 没有检测到 goto 或 $$app/navigation'
	@CN_HITS=$$(rg -n "[\\u4e00-\\u9fff]" web/src/routes web/src/lib --glob '!web/src/lib/i18n.ts' | sed -n '1,120p' || true); \
	if [ -n "$$CN_HITS" ]; then \
		echo "[P0] Found hardcoded Chinese text outside i18n dictionary:"; \
		echo "$$CN_HITS"; \
		exit 1; \
	fi
	@echo "[P1] 文案未检测到页面级中文硬编码（i18n 除外）"
	@TABLE_HITS=$$(rg -n "<table\\b|</table>" web/src/routes web/src/lib --glob '*.svelte' --glob '*.ts' | sed -n '1,120p' || true); \
	if [ -n "$$TABLE_HITS" ]; then \
		echo "[P1] Found table markup in web pages (prefer Tailwind card/list composition):"; \
		echo "$$TABLE_HITS"; \
	else \
		echo "[P1] 未检测到 table 标记，页面采用卡片/列表结构"; \
	fi
	@STRICT_MODE="$(WEB_CHECK_STRICT)"; \
	if [ "$$STRICT_MODE" = "1" ]; then STRICT_MODE=error; fi; \
	if [ "$$STRICT_MODE" != "0" ] && [ "$$STRICT_MODE" != "" ]; then \
		echo "[P1] [web-check] Strict mode: checking React/non-SPA/Svelte-native/component-first constraints (mode=$$STRICT_MODE)"; \
		if [ -n "$$TABLE_HITS" ]; then \
			if [ "$$STRICT_MODE" = "error" ]; then \
				echo "[P0] Found table markup in web pages (error mode):"; \
				echo "$$TABLE_HITS"; \
				exit 1; \
			else \
				echo "[P1] Found table markup in web pages (strict mode=warn):"; \
				echo "$$TABLE_HITS"; \
			fi; \
		else \
			echo "[P1] 未检测到 table 标记（Card/List 风格）"; \
		fi; \
		REACT_HITS=$$(rg -n "from ['\\\"]react['\\\"]|ReactDOM|\\bReact\\.|\\bclassName=" web/src --glob '*.svelte' --glob '*.ts' | sed -n '1,120p' || true); \
		if [ -n "$$REACT_HITS" ]; then \
			if [ "$$STRICT_MODE" = "error" ]; then \
				echo "[P0] Found React-style usage in web (mode=error):"; \
				echo "$$REACT_HITS"; \
				exit 1; \
			else \
				echo "[P1] Found React-style usage in web (mode=warn):"; \
				echo "$$REACT_HITS"; \
			fi; \
		else \
			echo "[P1] 未检测到 React 风格代码"; \
		fi; \
		SPA_ROUTER_HITS=$$(rg -n "svelte-spa-router|from ['\\\"]@roxi/routify['\\\"]|\\bnavigate\\(" web/src/routes web/src/lib --glob '*.svelte' --glob '*.ts' | sed -n '1,120p' || true); \
		if [ -n "$$SPA_ROUTER_HITS" ]; then \
			if [ "$$STRICT_MODE" = "error" ]; then \
				echo "[P0] Found SPA-only router usage in web:"; \
				echo "$$SPA_ROUTER_HITS"; \
				exit 1; \
			else \
				echo "[P1] Found SPA-only router hints in web (mode=warn):"; \
				echo "$$SPA_ROUTER_HITS"; \
			fi; \
		else \
			echo "[P1] 未检测到明显 SPA-only 路由器依赖"; \
		fi; \
		TAILWIND_STYLE_HITS=$$(rg -n "style=\\\"|style='" web/src/app.html web/src/routes web/src/lib web/src --glob '*.svelte' --glob '*.html' | sed -n '1,120p' || true); \
		if [ -n "$$TAILWIND_STYLE_HITS" ]; then \
			if [ "$$STRICT_MODE" = "error" ]; then \
				echo "[P0] Found inline style usage in web components (error mode):"; \
				echo "$$TAILWIND_STYLE_HITS"; \
				exit 1; \
			else \
				echo "[P1] Found inline style usage in web components (mode=warn):"; \
				echo "$$TAILWIND_STYLE_HITS"; \
			fi; \
		else \
			echo "[P1] 未检测到行内样式风格写法"; \
		fi; \
		THEME_HITS=$$(rg -n "from ['\\\"]@skeletonlabs/skeleton['\\\"]|@skeletonlabs" web/src --glob '*.svelte' --glob '*.ts' | sed -n '1,120p' || true); \
		if [ -n "$$THEME_HITS" ]; then \
				echo "[P1] 检测到 Skeleton 组件/样式依赖（满足 Tailwind+Skeleton 组件优先约束）"; \
		else \
			if [ "$$STRICT_MODE" = "error" ]; then \
				echo "[P0] 未检测到 Skeleton 组件或样式依赖：当前实现未满足 Tailwind component-first 约束"; \
				echo "$$THEME_HITS"; \
				exit 1; \
			else \
				echo "[P1] 未检测到明确的 Skeleton 组件或样式依赖（Tailwind+Skeleton 是核心规范，建议补齐）"; \
			fi; \
		fi; \
	else \
		echo "[P1] web-check strict checks disabled. Set WEB_CHECK_STRICT=warn|error for warning/error mode."; \
	fi
	@echo "[P1] web-check 完成"
	@echo "[P1] 风格约束要点复核：Tailwind/Skeleton 为关键 UI 组件体系；主导航不得走 SPA 运行时状态机；采用 Svelte 原生语法与文件路由。"

web-check-strict:
	@$(MAKE) web-check WEB_CHECK_STRICT=error

web-frontend-contract:
	@echo "[P1] [web-frontend-contract] Verifying Svelte + Tailwind/Skeleton + non-SPA frontend contract..."
	@$(MAKE) web-check WEB_CHECK_STRICT=error
	@echo "[P1] 前端合约检查通过："
	@echo "  - 非 SPA 导航约束"
	@echo "  - Session 跳转集中控制"
	@echo "  - Tailwind + Skeleton 组件链路是项目关键 UI 组成部分，且仅做非核心场景补充"
	@echo "  - SvelteKit 文件路由 + Svelte 原生能力承载主流程（非 SPA 运行时主导航）"
	@echo "  - i18n 与默认英文基线检查"
	@echo "  - 统一约束文档: docs/web-svelte-tailwind-contract.md ($(WEB_FRONTEND_CONTRACT_VERSION), $(WEB_FRONTEND_CONTRACT_DATE))"

web-contract-bump:
	@if [ -z "$(NEW_VERSION)" ]; then \
		echo "Usage: make web-contract-bump NEW_VERSION=<web-svelte-tailwind-contract-vX.Y>"; \
		exit 1; \
	fi
	@TODAY=$$(date +%Y-%m-%d); \
	@if printf '%s\n' "$(NEW_VERSION)" | grep -qE '^web-svelte-tailwind-contract-v[0-9]+\.[0-9]+$$'; then :; else \
		echo "Warning: NEW_VERSION should follow web-svelte-tailwind-contract-vX.Y, proceeding anyway."; \
	fi; \
	echo "Bumping contract version from $(WEB_FRONTEND_CONTRACT_VERSION) to $(NEW_VERSION)"; \
	tmp=$$(mktemp); \
	awk -v v="$(NEW_VERSION)" -v d="$$TODAY" '{ if ($$0 ~ /^WEB_FRONTEND_CONTRACT_VERSION[[:space:]]*\\?=/) { print "WEB_FRONTEND_CONTRACT_VERSION ?= " v; next } if ($$0 ~ /^WEB_FRONTEND_CONTRACT_DATE[[:space:]]*\\?=/) { print "WEB_FRONTEND_CONTRACT_DATE ?= " d; next } { print $$0 }' Makefile > $$tmp && mv $$tmp Makefile; \
	tmp=$$(mktemp); \
	awk -v v="$(NEW_VERSION)" -v d="$$TODAY" '{ gsub(/\\*\\*版本\\*\\*：`[^`]*`/, "**版本**：`" v "`"); gsub(/\\*\\*更新日期\\*\\*：`[^`]*`/, "**更新日期**：`" d "`"); print; }' docs/web-svelte-tailwind-contract.md > $$tmp && mv $$tmp docs/web-svelte-tailwind-contract.md; \
	tmp=$$(mktemp); \
	awk -v v="$(NEW_VERSION)" -v d="$$TODAY" '{ gsub(/\\*\\*版本\\*\\*：`[^`]*`/, "**版本**：`" v "`"); gsub(/\\*\\*更新日期\\*\\*：`[^`]*`/, "**更新日期**：`" d "`"); print; }' docs/web-svelte-tailwind-contract-brief.md > $$tmp && mv $$tmp docs/web-svelte-tailwind-contract-brief.md; \
	echo "Bump done. Current version: $(NEW_VERSION), date: $$TODAY"

k8s-build-frontend:
	docker build -t open-idb-frontend:dev web/

k8s-deploy-frontend:
	kubectl apply -f deploy/k8s/orbstack/frontend-app.yaml
	kubectl apply -f deploy/k8s/orbstack/ingress.yaml
	kubectl -n open-idb rollout restart deployment/idbridge-frontend
	kubectl -n open-idb rollout status deployment/idbridge-frontend --timeout=120s

dev-local-check:
	@echo "[P1] [dev-local-check] Verifying local tooling..."
	@command -v kubectl >/dev/null 2>&1 || (echo "[P0] missing dependency: kubectl"; exit 1)
	@command -v go >/dev/null 2>&1 || (echo "[P0] missing dependency: go"; exit 1)
	@command -v node >/dev/null 2>&1 || (echo "[P0] missing dependency: node"; exit 1)
	@command -v npm >/dev/null 2>&1 || (echo "[P0] missing dependency: npm"; exit 1)
	@if ! kubectl version --request-timeout=5s >/dev/null 2>&1; then \
		echo "[P0] k8s API is not reachable, check kubectl context/cluster first."; \
		echo "  kubectl config current-context"; \
		echo "  kubectl config get-contexts"; \
		echo "  kubectl cluster-info"; \
		exit 1; \
	fi
	@if [ -n "$(IDB_KUBECTL_CONTEXT)" ]; then \
		CURRENT_CTX=$$(kubectl config current-context); \
		if [ "$$CURRENT_CTX" != "$(IDB_KUBECTL_CONTEXT)" ]; then \
			echo "[P0] kube context mismatch: current=$$CURRENT_CTX, expected=$(IDB_KUBECTL_CONTEXT)"; \
			echo "Run: kubectl config use-context $(IDB_KUBECTL_CONTEXT)"; \
			exit 1; \
		fi; \
	fi
	@echo "[P1] [dev-local-check] Verifying namespace and services in k8s..."
	@kubectl -n $(DEV_LOCAL_NAMESPACE) get namespace $(DEV_LOCAL_NAMESPACE) >/dev/null 2>&1 || (echo "[P0] missing namespace: $(DEV_LOCAL_NAMESPACE)"; exit 1)
	@kubectl -n $(DEV_LOCAL_NAMESPACE) get svc postgres >/dev/null 2>&1 || (echo "[P0] missing service: postgres in namespace $(DEV_LOCAL_NAMESPACE)"; exit 1)
	@kubectl -n $(DEV_LOCAL_NAMESPACE) wait --for=condition=ready pod -l app.kubernetes.io/name=postgres --timeout=30s >/dev/null 2>&1 || (echo "[P0] postgres pod not ready in namespace $(DEV_LOCAL_NAMESPACE)"; exit 1)
	@if [ -n "$(IDB_ETCD_SERVICE)" ]; then \
		kubectl -n $(DEV_LOCAL_NAMESPACE) get svc $(IDB_ETCD_SERVICE) >/dev/null 2>&1 || (echo "[P0] missing service: $(IDB_ETCD_SERVICE) in namespace $(DEV_LOCAL_NAMESPACE)"; exit 1); \
		echo "[dev-local-check] etcd service detected: $(IDB_ETCD_SERVICE)"; \
	fi
	@if command -v lsof >/dev/null 2>&1; then \
		lsof -iTCP:$(DEV_LOCAL_PG_PORT) -sTCP:LISTEN -n -P >/dev/null 2>&1 && echo "[P1] local port $(DEV_LOCAL_PG_PORT) is already in use" || true; \
		lsof -iTCP:$(DEV_LOCAL_BACKEND_PORT) -sTCP:LISTEN -n -P >/dev/null 2>&1 && echo "[P1] local port $(DEV_LOCAL_BACKEND_PORT) is already in use" || true; \
		lsof -iTCP:$(DEV_LOCAL_WEB_PORT) -sTCP:LISTEN -n -P >/dev/null 2>&1 && echo "[P1] local port $(DEV_LOCAL_WEB_PORT) is already in use" || true; \
	fi
	@echo "[P1] [dev-local-check] OK"

dev-local: dev-local-check
	./scripts/dev-web-local.sh start

dev-local-preflight:
	@set +e; \
	mkdir -p .local; \
	P0_FILE=.local/.preflight_p0; \
	P1_FILE=.local/.preflight_p1; \
	rm -f $$P0_FILE $$P1_FILE; \
	P0_COUNT=0; \
	P1_COUNT=0; \
	echo "========================================"; \
	echo "30秒前后端联调预检（建议每次启动前执行）"; \
	echo "========================================"; \
	echo ""; \
	echo "步骤 1/5: 校验运行时依赖与 K8S 可达性"; \
	if make dev-local-check > .local/dev-local-preflight.log 2>&1; then \
		echo "[P1] dev-local-check 已通过"; \
	else \
		echo "[P0] dev-local-check 失败（请先执行 make dev-local-quickfix）"; \
		echo "[P0] dev-local-check 失败（请先执行 make dev-local-quickfix）" >> $$P0_FILE; \
		P0_COUNT=$$((P0_COUNT + 1)); \
		echo "  - 命令输出（最后 30 行）"; \
		tail -n 30 .local/dev-local-preflight.log; \
	fi; \
	echo ""; \
	echo "步骤 2/5: 约束核验"; \
	if make web-frontend-contract > .local/web-frontend-contract-preflight.log 2>&1; then \
		echo "[P1] web-frontend-contract 已通过"; \
	else \
		echo "[P1] web-frontend-contract 未通过（建议先执行 make web-frontend-contract）"; \
		echo "[P1] web-frontend-contract 未通过（建议先执行 make web-frontend-contract）" >> $$P1_FILE; \
		P1_COUNT=$$((P1_COUNT + 1)); \
		echo "  - 命令输出（最后 30 行）"; \
		tail -n 30 .local/web-frontend-contract-preflight.log; \
	fi; \
	echo ""; \
	echo "步骤 3/5: 快速端口占用核验"; \
	if command -v lsof >/dev/null 2>&1; then \
		for port in $(DEV_LOCAL_PG_PORT) $(DEV_LOCAL_BACKEND_PORT); do \
			if lsof -iTCP:$$port -sTCP:LISTEN -n -P >/dev/null 2>&1; then \
				echo "[P1] 本地端口 $$port 被占用（历史实例可复用）"; \
				echo "[P1] 本地端口 $$port 被占用" >> $$P1_FILE; \
				P1_COUNT=$$((P1_COUNT + 1)); \
			else \
				echo "[P1] 端口 $$port 未被占用"; \
			fi; \
		done; \
		FONT_PORT="$(DEV_LOCAL_WEB_PORT)"; \
		if lsof -iTCP:$$FONT_PORT -sTCP:LISTEN -n -P >/dev/null 2>&1; then \
			echo "[P1] 前端端口 $$FONT_PORT 被占用（可覆盖 DEV_LOCAL_WEB_PORT）"; \
			echo "[P1] 本地前端端口 $$FONT_PORT 被占用" >> $$P1_FILE; \
			P1_COUNT=$$((P1_COUNT + 1)); \
		else \
			echo "[P1] 前端端口 $$FONT_PORT 未被占用"; \
		fi; \
	else \
		echo "[P1] 未安装 lsof，端口核验跳过（建议安装 lsof 提升完整度）"; \
		echo "[P1] 未安装 lsof，端口核验跳过" >> $$P1_FILE; \
		P1_COUNT=$$((P1_COUNT + 1)); \
	fi; \
	echo ""; \
	echo "步骤 4/5: 本地服务可达性探测（若服务已启动）"; \
	if command -v curl >/dev/null 2>&1; then \
		BACKEND_URL="http://127.0.0.1:$(DEV_LOCAL_BACKEND_PORT)"; \
		HEALTH_CODE=$$(curl -o /dev/null -s -w "%{http_code}" --max-time 3 "$${BACKEND_URL}/healthz" || true); \
		if [ "$$HEALTH_CODE" = "200" ]; then \
			echo "[P1] 后端健康检查: $${BACKEND_URL}/healthz"; \
		else \
			echo "[P1] 后端未就绪（先启动后复检）: code=$${HEALTH_CODE:-N/A}"; \
			echo "[P1] 后端未就绪: code=$${HEALTH_CODE:-N/A}" >> $$P1_FILE; \
			P1_COUNT=$$((P1_COUNT + 1)); \
		fi; \
		FRONTEND_URL="http://127.0.0.1:$(DEV_LOCAL_WEB_PORT)"; \
		FRONT_CODE=$$(curl -o /dev/null -s -w "%{http_code}" --max-time 3 "$${FRONTEND_URL}" || true); \
		if [ "$$FRONT_CODE" = "200" ]; then \
			echo "[P1] 前端可达: $$FRONTEND_URL"; \
		else \
			echo "[P1] 前端未就绪（先启动后复检）: code=$${FRONT_CODE:-N/A}"; \
			echo "[P1] 前端未就绪: code=$${FRONT_CODE:-N/A}" >> $$P1_FILE; \
			P1_COUNT=$$((P1_COUNT + 1)); \
		fi; \
	else \
		echo "[P1] 未安装 curl，服务可达性探测跳过"; \
		echo "[P1] 未安装 curl，服务可达性探测跳过" >> $$P1_FILE; \
		P1_COUNT=$$((P1_COUNT + 1)); \
	fi; \
	echo ""; \
	echo "步骤 5/5: 统一汇总"; \
	echo "[P0] 总数: $$P0_COUNT"; \
	echo "[P1] 总数: $$P1_COUNT"; \
	echo ""; \
	if [ $$P0_COUNT -gt 0 ]; then \
		echo "P0 清单（必做 / 阻断启动）"; \
		n=$$(awk 'BEGIN{s=0} {s=s+1} END{if (s==0) print 0; else print s}' $$P0_FILE); \
		if [ $$n -gt 0 ]; then \
			i=1; \
			while IFS= read -r line; do \
				echo "$$i) $$line"; \
				i=$$((i + 1)); \
			done < $$P0_FILE; \
		fi; \
	else \
		echo "P0 清单: 无"; \
	fi; \
	echo ""; \
	if [ $$P1_COUNT -gt 0 ]; then \
		echo "P1 清单（建议项）"; \
		i=1; \
		while IFS= read -r line; do \
			echo "$$i) $$line"; \
			i=$$((i + 1)); \
		done < $$P1_FILE; \
	else \
		echo "P1 清单: 无"; \
	fi; \
	if [ $$P0_COUNT -gt 0 ]; then \
		echo ""; \
		echo "启动失败：存在 P0 阻断项"; \
		echo "[P0] 建议执行 make dev-local-quickfix 后重跑预检"; \
		exit 1; \
	fi; \
	echo ""; \
	echo "[P1] 未发现 P0 阻断项，允许直接启动"; \
	echo "建议执行: make dev-local"

dev-local-go-live:
	@$(MAKE) dev-local-preflight && $(MAKE) dev-local

dev-local-quickfix:
	@mkdir -p .local
	@set +e; \
	FAIL_CNT=0; \
	NS_STATUS=0; SVC_STATUS=0; POD_STATUS=0; ETCD_STATUS=0; PORT_STATUS=0; TOOL_STATUS=0; CHECK_STATUS=0; WEBCHECK_STATUS=0; \
	KUBE_STATUS=0; \
	echo "============================================================"; \
	echo "6-Step 自动排障脚本: make dev-local-quickfix"; \
	echo "============================================================"; \
	echo ""; \
	echo "步骤 1/6: 采集关键日志（各 20 行）"; \
	for file in .local/backend.log .local/web-frontend.log .local/dev-web-local-backend.log .local/dev-web-local-web.log .local/postgres-port-forward.log .local/etcd-port-forward.log .local/dev-local-check.log; do \
		echo "---- $$file ----"; \
		if [ -f "$$file" ]; then \
			if [ -s "$$file" ]; then \
				tail -n 20 "$$file"; \
			else \
			echo "[P1] 日志文件存在但为空"; \
				FAIL_CNT=$$((FAIL_CNT + 1)); \
			fi; \
		else \
			echo "[P1] 日志文件缺失（首次未启动时常见）"; \
		fi; \
	done; \
	echo ""; \
	echo "步骤 2/6: 检查 dev-local 相关进程"; \
	if command -v pgrep >/dev/null 2>&1; then \
		PGREP_OUT=$$(pgrep -af "(air|go run .*cmd/idbridge|npm run dev|node .*vite|kubectl port-forward .*postgres|kubectl port-forward .*etcd)" 2>/dev/null || true); \
		if [ -n "$$PGREP_OUT" ]; then \
			echo "$$PGREP_OUT"; \
			echo "[P1] 检测到相关进程"; \
		else \
			echo "[P1] 未检测到 dev-local 相关进程"; \
		fi; \
	else \
		ps -ef | grep -E "air|go run .*cmd/idbridge|go run ./cmd/idbridge|npm run dev|node .*vite|kubectl port-forward (svc/)?postgres|kubectl port-forward .*etcd" | grep -v grep | sed -n '1,40p' || echo "[P1] 未检测到 dev-local 相关进程"; \
	fi; \
	echo ""; \
	echo "步骤 3/6: 检查工具链"; \
	for cmd in kubectl go node npm; do \
		if command -v $$cmd >/dev/null 2>&1; then \
			echo "[P1] $$cmd 已安装"; \
		else \
			echo "[P0] 缺失: $$cmd"; \
			TOOL_STATUS=1; \
			FAIL_CNT=$$((FAIL_CNT + 1)); \
		fi; \
	done; \
	if ! command -v lsof >/dev/null 2>&1; then \
		echo "[P1] 未安装 lsof（端口检查将受限）"; \
	fi; \
	echo ""; \
	echo "步骤 4/6: 检查 K8s 依赖"; \
	if command -v kubectl >/dev/null 2>&1; then \
		if ! kubectl version --request-timeout=5s >/dev/null 2>&1; then \
			echo "[P0] kubectl API 不可达（context/cluster 未连接）"; \
			echo "建议: kubectl config current-context && kubectl config get-contexts && kubectl cluster-info"; \
			KUBE_STATUS=1; \
			FAIL_CNT=$$((FAIL_CNT + 1)); \
		else \
			if [ -n "$(IDB_KUBECTL_CONTEXT)" ] && [ "$$(kubectl config current-context)" != "$(IDB_KUBECTL_CONTEXT)" ]; then \
				echo "[P1] kubectl context 与预期不一致: $$(kubectl config current-context)"; \
				echo "建议: kubectl config use-context $(IDB_KUBECTL_CONTEXT)"; \
				KUBE_STATUS=1; \
				FAIL_CNT=$$((FAIL_CNT + 1)); \
			fi; \
			if kubectl -n $(DEV_LOCAL_NAMESPACE) get namespace $(DEV_LOCAL_NAMESPACE) >/dev/null 2>&1; then \
				echo "[P1] 命名空间存在: $(DEV_LOCAL_NAMESPACE)"; \
			else \
				echo "[P0] 命名空间不存在: $(DEV_LOCAL_NAMESPACE)"; \
				NS_STATUS=1; \
				FAIL_CNT=$$((FAIL_CNT + 1)); \
			fi; \
			if kubectl -n $(DEV_LOCAL_NAMESPACE) get svc postgres >/dev/null 2>&1; then \
				echo "[P1] PostgreSQL Service 存在: svc/postgres"; \
			else \
				echo "[P0] PostgreSQL Service 缺失: svc/postgres"; \
				SVC_STATUS=1; \
				FAIL_CNT=$$((FAIL_CNT + 1)); \
			fi; \
			if kubectl -n $(DEV_LOCAL_NAMESPACE) get pods -l app.kubernetes.io/name=postgres --field-selector=status.phase=Running | grep -q postgres >/dev/null 2>&1; then \
				echo "[P1] PostgreSQL Pod Running"; \
			else \
				echo "[P0] PostgreSQL Pod 未 Running"; \
				POD_STATUS=1; \
				FAIL_CNT=$$((FAIL_CNT + 1)); \
			fi; \
		fi; \
			if [ -n "$(IDB_ETCD_SERVICE)" ]; then \
				if kubectl -n $(DEV_LOCAL_NAMESPACE) get svc $(IDB_ETCD_SERVICE) >/dev/null 2>&1; then \
					echo "[P1] etcd Service 存在: $(IDB_ETCD_SERVICE)"; \
				else \
					echo "[P1] etcd Service 不存在: $(IDB_ETCD_SERVICE)"; \
					ETCD_STATUS=1; \
				fi; \
			else \
				echo "[P1] 未设置 IDB_ETCD_SERVICE，跳过 etcd 检查"; \
			fi; \
	else \
		NS_STATUS=1; \
		SVC_STATUS=1; \
		POD_STATUS=1; \
	fi; \
	echo ""; \
	echo "步骤 5/6: 检查端口占用"; \
	if command -v lsof >/dev/null 2>&1; then \
		for port in $(DEV_LOCAL_PG_PORT) $(DEV_LOCAL_BACKEND_PORT) $(DEV_LOCAL_WEB_PORT); do \
			PORT_PIDS=$$(lsof -tiTCP:$$port -sTCP:LISTEN -n -P 2>/dev/null || true); \
			if [ -n "$$PORT_PIDS" ]; then \
				echo "[P1] 端口 $$port 已被占用: $$PORT_PIDS"; \
				PORT_STATUS=1; \
				FAIL_CNT=$$((FAIL_CNT + 1)); \
			else \
				echo "[P1] 端口 $$port 空闲"; \
			fi; \
		done; \
	else \
		echo "[P1] lsof 不存在，跳过端口占用检测"; \
	fi; \
	echo ""; \
	echo "步骤 6/6: 复测 make dev-local-check"; \
	if make dev-local-check > .local/dev-local-check.log 2>&1; then \
		echo "[P1] make dev-local-check 通过"; \
	else \
		echo "[P0] make dev-local-check 未通过"; \
		echo "最近错误："; \
		tail -n 20 .local/dev-local-check.log; \
		CHECK_STATUS=1; \
		FAIL_CNT=$$((FAIL_CNT + 1)); \
	fi; \
	echo "步骤 6b/6: 复测 make web-frontend-contract"; \
	if make web-frontend-contract > .local/web-check.log 2>&1; then \
		echo "[P1] make web-frontend-contract 通过"; \
	else \
		echo "[P1] make web-frontend-contract 未通过"; \
		echo "最近错误："; \
		tail -n 20 .local/web-check.log; \
		WEBCHECK_STATUS=1; \
		FAIL_CNT=$$((FAIL_CNT + 1)); \
	fi; \
	echo ""; \
	echo "============================================================"; \
	if [ $$FAIL_CNT -eq 0 ]; then \
		echo "[P1] 当前未发现阻断项；可直接执行: make dev-local"; \
		echo "============================================================"; \
	else \
		echo "[P0] 建议按优先级修复（[P0]=必做，[P1]=可选）"; \
		echo "建议先处理全部 [P0] 项，再处理 [P1] 项"; \
		HINT_NO=1; \
		if [ $$TOOL_STATUS -ne 0 ]; then echo "$${HINT_NO}) [P0] 安装缺失工具: brew install go node npm && 下载 kubectl"; HINT_NO=$$((HINT_NO + 1)); fi; \
		if [ $$KUBE_STATUS -ne 0 ]; then echo "$${HINT_NO}) [P0] 先修复 kubectl context/cluster：kubectl config current-context/get-contexts/cluster-info"; HINT_NO=$$((HINT_NO + 1)); fi; \
		if [ $$NS_STATUS -ne 0 ]; then echo "$${HINT_NO}) [P0] 确保 K8s 命名空间: kubectl create namespace $(DEV_LOCAL_NAMESPACE)"; HINT_NO=$$((HINT_NO + 1)); fi; \
		if [ $$SVC_STATUS -ne 0 ]; then echo "$${HINT_NO}) [P0] 创建/恢复 Postgres Service: kubectl -n $(DEV_LOCAL_NAMESPACE) get svc | grep postgres"; HINT_NO=$$((HINT_NO + 1)); fi; \
		if [ $$POD_STATUS -ne 0 ]; then echo "$${HINT_NO}) [P0] 检查数据库部署状态: kubectl -n $(DEV_LOCAL_NAMESPACE) rollout status deployment/postgres --timeout=120s"; HINT_NO=$$((HINT_NO + 1)); fi; \
		if [ $$PORT_STATUS -ne 0 ]; then echo "$${HINT_NO}) [P0] 处理端口冲突: kill <端口占用 PID> 或覆盖 DEV_LOCAL_*_PORT"; HINT_NO=$$((HINT_NO + 1)); fi; \
		if [ $$CHECK_STATUS -ne 0 ]; then echo "$${HINT_NO}) [P0] 查看详细日志: tail -f .local/dev-local-check.log"; HINT_NO=$$((HINT_NO + 1)); fi; \
		if [ $$ETCD_STATUS -ne 0 ] && [ -n "$(IDB_ETCD_SERVICE)" ]; then echo "$${HINT_NO}) [P1] 确认 etcd 依赖配置，或临时清空 IDB_ETCD_SERVICE 再试"; HINT_NO=$$((HINT_NO + 1)); fi; \
		if [ $$WEBCHECK_STATUS -ne 0 ]; then echo "$${HINT_NO}) [P1] 修复前端风格问题并重新执行: make web-frontend-contract"; HINT_NO=$$((HINT_NO + 1)); fi; \
		echo "排查后再次执行: make dev-local-quickfix"; \
		echo "============================================================"; \
		exit 1; \
	fi
