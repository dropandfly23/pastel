FROM alpine:3.8

ADD pastel /etc/pastel/pastel
ADD public /etc/pastel/public

RUN chmod 777 -R /etc/pastel
RUN apk add --no-cache ca-certificates

WORKDIR /etc/pastel

ENTRYPOINT [ "./pastel" ]

CMD ["start"]
