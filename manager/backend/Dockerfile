FROM golang:1.24-alpine AS builder

WORKDIR /app

# Use Chinese mirror for faster downloads; allow go.sum updates if incomplete
ENV GOPROXY=https://goproxy.cn,direct
ENV GOFLAGS=-mod=mod

# 复制go mod文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata git git-lfs
WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .
COPY --from=builder /app/config ./config

# 设置时区
ENV TZ=Asia/Shanghai

EXPOSE 8080

CMD ["./main"]