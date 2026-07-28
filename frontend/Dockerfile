FROM node:20-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM alpine:3.20
COPY --from=builder /app/dist /app/dist
CMD ["sh", "-c", "rm -rf /output/* && cp -r /app/dist/* /output/"]
