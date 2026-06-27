pnpm build
scp -r dist/ server:~/apps/img-app/
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o img-app-backend
scp backend/img-app-backend server:~/apps/img-app/

