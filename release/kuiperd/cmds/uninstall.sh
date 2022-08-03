#!/bin/bash
# 停止并卸载服务

SERVICE_NAME=${1:-"kuiperd"}
SERVICE_PATH="/usr/lib/systemd/system"

uninstall_service()
{
    if [ -f "$SERVICE_PATH/$1.service" ]; then
        systemctl stop $1
        systemctl disable $1 2>/dev/null2>/dev/null
        rm -f "$SERVICE_PATH/$1.service"
    fi
}

echo "uninstall $SERVICE_NAME starting."
uninstall_service $SERVICE_NAME
echo "uninstall $SERVICE_NAME end."
