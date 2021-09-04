# S3 Sidecar container

A container downloads a compressed file from S3 into a local directory (e.g `/data`) which can be a shared volume with other containers. In addition, it will check if the file has been modified in the bucket, if so, download it again.

An additional directory can be defined for updating the S3 file (e.g `/data/upload`), if a file is found in that directory for the given file key, it will upload the object to S3 if is newer (timestamp) than the upstream.


## Docker compose example


```yaml
version: "2"
services:
  telegrambot:
    image: mydict
    container_name: mydictbot
    volumes:
      - /home/pi/dictdata:/var/lib/mydict
    environment:
      - BADGERDB_DIR=/var/lib/mydict
      - BADGERDB_BCK_DIR=/var/bck/mydict
    restart: always
  dictdata:
    image: s3sidecar
    container_name: s3sidecar
    volumes:
      - /home/pi/dictdata:/data
      - ~/.aws/:/root/.aws:ro
    environment:
      - AWS_REGION=eu-west-1
      - S3_BUCKET=neckapps
      - S3_OBJECT=dictionary.txt
      - W_DIR=/data
      - U_DIR=/data/upload
      - INTERVAL=1800
    restart: always
```
