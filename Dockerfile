FROM debian:stable-slim

COPY youtube-custom-feeds /bin/youtube-custom-feeds

CMD ["/bin/youtube-custom-feeds"]