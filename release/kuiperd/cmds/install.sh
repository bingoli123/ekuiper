#!/bin/bash
# 安装并运行服务
BASEDIR=`dirname $0`
BASEDIR=`(cd "$BASEDIR"; pwd)`
PROGRAM=$BASEDIR/monitoring.sh
SERVICE_PATH="/usr/lib/systemd/system"
SERVICE_NAME=${1:-"kuiperd"}

create_service_file()
{
    file_path="$SERVICE_PATH/$1.service"
    echo "[Unit]" > $file_path
    echo "Description=$1" >> $file_path
    echo "After=network.target" >> $file_path
    echo "Before=crond.service" >> $file_path
    echo "[Service]" >> $file_path
    echo "User=root" >> $file_path
    echo "Group=root" >> $file_path
    echo "ExecStart=$BASEDIR/service/bin/$1 -c $BASEDIR/service/conf/config.yml" >> $file_path
    echo "[Install]" >> $file_path
    echo "WantedBy=multi-user.target" >> $file_path
}

install_service()
{
    create_service_file $SERVICE_NAME
    systemctl daemon-reload
    systemctl start $SERVICE_NAME
    systemctl enable $SERVICE_NAME 2>/dev/null
}

echo "install service $SERVICE_NAME starting."
install_service $SERVICE_NAME
echo "install service $SERVICE_NAME end."


