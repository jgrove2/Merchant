# frontend/Dockerfile
FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build
# Using 'preview' to serve the built app on port 5173
CMD ["npm", "run", "preview", "--", "--host"]
