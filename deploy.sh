#!/bin/bash
set -euo pipefail

pnpm build
scp -r dist/ server:~/apps/img-app/
( cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o img-app-backend . )
scp backend/img-app-backend backend/.env server:~/apps/img-app/
# 数据库位于后端工作目录下的 data/img-app.db（或由 APP_DB_PATH 指定），部署时不要覆盖或删除该文件。
echo "Files uploaded. Restart the remote backend process to activate the new binary."
