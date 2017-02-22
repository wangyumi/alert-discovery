FROM index.caicloud.io/debian:jessie

WORKDIR /
ADD alert-discovery /

# Set the timezone to Shanghai
RUN echo "Asia/Shanghai" > /etc/timezone
RUN dpkg-reconfigure -f noninteractive tzdata

ENTRYPOINT ["/alert-discovery"]
CMD ["-h"]
