FROM node:16.0.0

WORKDIR /usr/src/app
COPY package*.json ./

RUN npm install

COPY . .
EXPOSE 6008
CMD ["node", "."]