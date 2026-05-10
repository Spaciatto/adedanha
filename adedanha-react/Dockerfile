# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# Runtime stage - Nginx serves the static files
FROM nginx:alpine

# Copy built files into /adedanha/ subfolder so nginx root-based serving works
RUN mkdir -p /usr/share/nginx/html/adedanha
COPY --from=builder /app/dist/ /usr/share/nginx/html/adedanha/
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
