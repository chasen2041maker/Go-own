FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY projects/04-investment-community ./projects/04-investment-community

RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./projects/04-investment-community/reference/cmd/api && \
    CGO_ENABLED=0 go build -trimpath -o /out/migrate ./projects/04-investment-community/reference/cmd/migrate && \
    CGO_ENABLED=0 go build -trimpath -o /out/seed ./projects/04-investment-community/reference/cmd/seed

FROM alpine:3.22 AS api-runtime
RUN addgroup -S community && adduser -S -G community community
COPY --from=build /out/ /app/
USER community
EXPOSE 8084
ENTRYPOINT ["/app/api"]

FROM swaggerapi/swagger-ui:v5.29.5 AS swagger-assets

FROM nginxinc/nginx-unprivileged:1.29.1-alpine AS swagger
COPY --from=swagger-assets /usr/share/nginx/html /usr/share/nginx/html
COPY projects/04-investment-community/swagger-nginx.conf /etc/nginx/conf.d/default.conf
COPY projects/04-investment-community/contracts/openapi.yaml /usr/share/nginx/html/openapi.yaml
COPY projects/04-investment-community/swagger-initializer.js /usr/share/nginx/html/swagger-initializer.js
USER 101
