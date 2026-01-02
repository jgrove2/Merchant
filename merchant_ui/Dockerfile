# frontend/Dockerfile
FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build
# The build output is usually in .output or dist depending on framework.
# This project seems to use TanStack Start / Nitro.
# The server entry point is usually .output/server/index.mjs

EXPOSE 3000

# Start the production server
CMD ["node", ".output/server/index.mjs"]
