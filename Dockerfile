FROM alpine

RUN apk add --no-cache tftp-hpa

WORKDIR /app
COPY tftp ./
COPY config.yaml /etc/tftp/

EXPOSE 69

HEALTHCHECK --interval=5m --timeout=5s CMD ["tftp", "127.0.0.1", "69", "-c", "get", "health"]
CMD ["/app/tftp", "serve"]
