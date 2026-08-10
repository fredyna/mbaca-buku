FROM node:20-alpine AS builder

WORKDIR /app

ARG VITE_SUPABASE_URL
ARG VITE_SUPABASE_ANON_KEY
ENV VITE_SUPABASE_URL=$VITE_SUPABASE_URL
ENV VITE_SUPABASE_ANON_KEY=$VITE_SUPABASE_ANON_KEY

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM alpine:3.20
COPY --from=builder /app/dist /app/dist
CMD ["sh", "-c", "rm -rf /output/* && cp -r /app/dist/* /output/"]
