[ -d /export/jenkins/workspace/op_dockerfile/${namespace}/${deploy_env}/code/ ] && rm -rf /export/jenkins/workspace/op_dockerfile/${namespace}/${deploy_env}/code/
mkdir /export/jenkins/workspace/op_dockerfile/${namespace}/${deploy_env}/code/ -p
cp -af /export/jenkins/workspace/${JOB_NAME}/. /export/jenkins/workspace/op_dockerfile/${namespace}/${deploy_env}/code/
cd /export/jenkins/workspace/op_dockerfile/${namespace}/${deploy_env}

# 定义动态端口范围
LOWER_PORT=49152
UPPER_PORT=65500

# 随机生成一个端口号
function get_random_port() {
    echo $((LOWER_PORT + RANDOM % (UPPER_PORT - LOWER_PORT + 1)))
}
# 检查端口是否可用
function is_port_free() {
    local port=$1
    sudo lsof -i -P -n | grep -q ":$port " || return 0
    return 1
}
# 找到一个可用的随机端口
function find_free_port() {
    while true; do
        port=$(get_random_port)
        if is_port_free $port; then
            echo $port
            return 0
        fi
    done
}
# 分配一个可用端口
port=$(find_free_port)
echo ${port}

if [ ! -z ${env} ];then
  echo "非空"
  port=${port}
else
  port=${base_port}
  env=base
  dayuse=7200h
fi

if [ ! -z ${port} ];then
  echo "非空"
else
  port=${base_port}
  env=base
  dayuse=7200h
fi

echo ${deeplangenv}
BUILD_TIMESTAMP=$(date +%Y%m%d%H%M%S)
# 使用项目自带的多阶段 Dockerfile 构建镜像
# CODEUP_USER / CODEUP_PASSWORD 由 Jenkins 参数传入，用于拉取私有 Go 依赖
docker build \
  -f code/docker/Dockerfile \
  --build-arg CODEUP_USER=${CODEUP_USER} \
  --build-arg CODEUP_PASSWORD=${CODEUP_PASSWORD} \
  -t ${mirror_store}/lingo/${deploy_env}-${env}:${BUILD_TIMESTAMP} \
  code/

#创建日志目录
if [ ! -d /export/${jobname}-backend-log/${env}/logs ];then
  sudo mkdir -p /export/${jobname}-backend-log/${env}/logs
fi

cat > ${jobname}.sh << EOF
#!/bin/bash
docker run \
-itd \
-m 4G \
--cpus=1 \
-e MODE_ENV=test \
-v /export/${jobname}-backend-log/${env}/logs:/opt/output/logs \
--name=${deploy_env}-${env} \
${mirror_store}/lingo/${deploy_env}-${env}:${BUILD_TIMESTAMP}
EOF
chmod 755 ${jobname}.sh

#获取镜像
if [ ! -z "`docker ps -a | grep ${deploy_env}-${env} | awk '{print $1}'`" ];then old_image=`docker ps -a | grep ${deploy_env}-${env} | awk '{print $2}'`;fi
#删除容器
if [ ! -z "`docker ps -a | grep ${deploy_env}-${env} | awk '{print $1}'`" ];then  docker rm -f `docker ps -a | grep ${deploy_env}-${env} | awk '{print $1}'`;fi
if [ ! -z $old_image ];then docker rmi -f $old_image;fi

sleep 1

#启动容器（CMD ["./app"] 自动拉起服务，无需再 exec entrypoint）
bash ${jobname}.sh

sleep 1

#docker 容器名称
echo "-----------------------------------------"
echo "docker 容器名称为: ${deploy_env}-${env} "

# 判断是否走基准环境
if [ ${port} == ${base_port} ] && [ ${env} == "base" ] && [ ${dayuse} == "7200h" ];then
  echo "变量正确"
  exit 0
else
  echo "变量不正确"
fi

cd /export/jenkins/workspace/op_dockerfile/${namespace}/${deploy_env}
#备份原配置文件
if [ ! -z "`cat /etc/nginx/conf.d/${domain_name}.conf |grep -w "DeeplangLanEnv-${env}$"`" ];then sudo sed -i "/DeeplangLanEnv-${env}$/,+2 d" /etc/nginx/conf.d/${domain_name}.conf ;fi
if [ -z "`cat /etc/nginx/conf.d/${domain_name}.conf |grep -w "DeeplangLanEnv-${env}$"`" ];then sudo sed -i "/#deeplangenv/a\                if (\$http_env = "${env}"){ #DeeplangLanEnv-${env}\n                proxy_pass http://10.0.6.2:${port};\n            }" /etc/nginx/conf.d/${domain_name}.conf ;fi
#nginx生效
sudo nginx -t
sudo nginx -s reload
cd /export/jenkins/workspace/op_dockerfile/${namespace}/${deploy_env}/ && rm -rf code
