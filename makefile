server:
	hz update -idl ./idl/notice_webhook.thrift

init_server:
	hz new -module empyrean_lens

dev_start:
	export MODE_ENV=dev && go run *.go &

prod:
	MODE_ENV=prod bash ./run.sh &

pre:
	MODE_ENV=pre go run *.go &

# 前端：安装依赖
web_install:
	cd web && npm install

# 前端：本地开发（代理到 :19001）
web_dev:
	cd web && npm run dev

# 前端：构建产物（embed.FS 所需）
web_build:
	cd web && npm run build

# 构建 Docker 镜像（私有仓库认证通过环境变量 CODEUP_USER / CODEUP_PASSWORD 传入）
docker_build:
	docker build -f docker/Dockerfile \
		$(if $(CODEUP_USER),--build-arg CODEUP_USER=$(CODEUP_USER)) \
		$(if $(CODEUP_PASSWORD),--build-arg CODEUP_PASSWORD=$(CODEUP_PASSWORD)) \
		-t empyrean-lens:latest .


# ------------------------------
# common begin
# ------------------------------
zip:
	rm -rf output && \
	mkdir output && \
	GOOS=linux GOARCH=amd64 go build -o output/main $(MAIN_PATH) && \
	cp -r conf output && \
	(cd output && zip -r ../$(ZIP_NAME) .) && \
	rm -rf output
# ------------------------------
# common end
# ------------------------------

fc_cron:
	$(MAKE) zip ZIP_NAME=fc_cron.zip MAIN_PATH=fc/cron/main.go
