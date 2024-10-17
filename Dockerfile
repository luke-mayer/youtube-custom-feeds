# New
FROM debian:stable-slim

RUN apt-get update && apt-get install -y ca-certificates

COPY youtube-custom-feeds /bin/youtube-custom-feeds

CMD ["/bin/youtube-custom-feeds"]