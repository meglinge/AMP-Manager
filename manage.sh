#!/bin/bash

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

show_menu() {
    echo ""
    echo "=============================="
    echo "   AMP Manager 管理脚本"
    echo "=============================="
    echo "1. 启动服务"
    echo "2. 停止服务"
    echo "3. 更新并重启 (拉取代码 + 重新构建)"
    echo "4. 查看日志"
    echo "5. 查看状态"
    echo "0. 退出"
    echo "=============================="
}

start_service() {
    echo -e "${GREEN}正在启动服务...${NC}"
    docker-compose up -d
    echo -e "${GREEN}服务已启动${NC}"
}

stop_service() {
    echo -e "${YELLOW}正在停止服务...${NC}"
    docker-compose down
    echo -e "${YELLOW}服务已停止${NC}"
}

update_and_restart() {
    echo -e "${GREEN}正在从远端拉取最新代码...${NC}"
    git pull origin main || git pull origin master
    
    echo -e "${GREEN}正在停止旧服务...${NC}"
    docker-compose down
    
    echo -e "${GREEN}正在重新构建镜像...${NC}"
    docker-compose build --no-cache
    
    echo -e "${GREEN}正在启动新服务...${NC}"
    docker-compose up -d
    
    echo -e "${GREEN}更新完成！${NC}"
}

show_logs() {
    echo -e "${GREEN}显示日志 (Ctrl+C 退出)...${NC}"
    docker-compose logs -f
}

show_status() {
    echo -e "${GREEN}服务状态:${NC}"
    docker-compose ps
}

main() {
    while true; do
        show_menu
        read -p "请选择操作 [0-5]: " choice
        
        case $choice in
            1)
                start_service
                ;;
            2)
                stop_service
                ;;
            3)
                update_and_restart
                ;;
            4)
                show_logs
                ;;
            5)
                show_status
                ;;
            0)
                echo -e "${GREEN}再见！${NC}"
                exit 0
                ;;
            *)
                echo -e "${RED}无效选项，请重新选择${NC}"
                ;;
        esac
    done
}

main
