# LLM Server

```
export LLM_SERVER_REGISTRATION_SECRET="abc123"
go run *.go
```


## curl

```bash
curl 0.0.0.0:8080/status/1

curl -X POST 0.0.0.0:8080/job \
-F 'prompt=yolo'
```