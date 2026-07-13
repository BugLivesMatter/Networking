FROM node:22-alpine AS builder
WORKDIR /app
ARG VITE_NEUROOPS_MODE=api
ARG VITE_CLUSTER_API_URL=
ENV VITE_NEUROOPS_MODE=$VITE_NEUROOPS_MODE
ENV VITE_CLUSTER_API_URL=$VITE_CLUSTER_API_URL
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
