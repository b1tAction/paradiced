# Development stage - installs all deps, runs ts-node-dev for live reload
FROM node:20-alpine AS dev
WORKDIR /app
COPY package.json ./
RUN npm install
COPY tsconfig.json ./
COPY src/ ./src/
EXPOSE 2567
CMD ["npm", "run", "dev"]

# Production stage - installs production deps only, builds and runs compiled code
FROM node:20-alpine AS prod
WORKDIR /app
COPY package.json ./
RUN npm install --production
COPY tsconfig.json ./
COPY src/ ./src/
RUN npm run build
EXPOSE 2567
CMD ["node", "lib/app.js"]