# Cybertrap in Go

## Introduction

Backend is powered by 3 microservices written in Go, using MongoDB as its database.

- Domains Api: Provides CRD for domains
- Artefact Api: Provides CRD for artefacts
- Nuclei Scanning Service: Nuclei domain scanner

## Environment Setup
Each microservice have their own envs


Create the following envs in same level of `backend`
- `.env.domains` 
- `.env.artefact` 
- `.env.nuclei` 

TODO: Write the envs


### Run the go app
Once the envs are there we can build and run our app

```
go build -o {output-binary-name} ./cmd/{service-you-wanna-build}/main.go

-------
example
-------

go build -o domains-api ./cmd/domains_api/main.go
```

Then you can run each app by running the binary
```
./{output-binary-name} 

-------
example
-------

./domains-api
```

### Run using docker
We can also build our app and run in by using docker, the dockerfile is already included and you can see how the app is built by using docker.

We can trigger the build an run it by using following command
```
docker build -t domains-api:v1 --build-arg SERVICE_NAME=domains-api .
```

Then afterwards you can run it by using the following command 
```
docker run -p 8080:8080 domains-api/cinema-movies:v1
```

We will discuss more on what are the arguments and flags involved in the docker section.

---

## Docker compose
### Multi architecture build
```
docker buildx bake -f docker-compose.yml --push
```
